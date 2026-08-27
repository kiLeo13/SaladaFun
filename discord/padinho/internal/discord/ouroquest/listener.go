package ouroquest

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	appouroquest "github.com/kiLeo13/SaladaFun/discord/padinho/internal/application/ouroquest"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/locale/ptbr"
)

const (
	commandCorrelationTTL = 2 * time.Second
	defaultGameTimeout    = 3 * time.Minute
	gameUpdateBuffer      = 12
)

// Messenger owns the reply lifecycle required by the $oq feature.
type Messenger interface {
	SendReply(channelID, guildID, sourceMessageID string, components []discordgo.MessageComponent) (string, error)
	EditMessage(channelID, messageID string, components []discordgo.MessageComponent) error
	DeleteMessage(channelID, messageID string) error
}

// PreferenceService resolves and toggles per-user automatic assistance.
type PreferenceService interface {
	AutoMudaeOQ(userID uint64) (bool, error)
	ToggleAutoMudaeOQ(userID uint64) (bool, error)
}

// MessageLoader retrieves the current state of a replied-to Discord message.
type MessageLoader interface {
	LoadMessage(channelID, messageID string) (*discordgo.Message, error)
}

// EmojiIDs maps every semantic $oq color to its configured custom emoji ID.
type EmojiIDs struct {
	Blue, Teal, Green, Yellow, Orange, Purple, Red string
}

type pendingCommand struct {
	recorded  time.Time
	userID    uint64
	messageID string
}
type boardSnapshot struct {
	board       appouroquest.Board
	unavailable [appouroquest.CellCount]bool
	fingerprint string
	revealed    int
	terminal    bool
}
type gameSession struct {
	sourceID, channelID, guildID string
	updates                      chan boardSnapshot
	stop                         chan struct{}
	stopOnce                     sync.Once
}

// Listener correlates $oq commands, parses boards, and owns helper sessions.
type Listener struct {
	mudaeID     string
	emojiColors map[string]appouroquest.Color
	messenger   Messenger
	preferences PreferenceService
	messages    MessageLoader
	logger      *slog.Logger
	gameTimeout time.Duration
	mu          sync.Mutex
	ctx         context.Context
	pending     map[string][]pendingCommand
	games       map[string]*gameSession
}

// New constructs a validated Mudae $oq listener.
func New(mudaeID string, emojis EmojiIDs, messenger Messenger, preferences PreferenceService, messages MessageLoader, logger *slog.Logger) (*Listener, error) {
	if parsed, err := strconv.ParseUint(mudaeID, 10, 64); err != nil || parsed == 0 {
		return nil, errors.New("Mudae bot ID is invalid")
	}
	if messenger == nil || preferences == nil || messages == nil || logger == nil {
		return nil, errors.New("ouroquest dependency is nil")
	}
	colors := make(map[string]appouroquest.Color, 7)
	configured := []struct {
		name, id string
		color    appouroquest.Color
	}{
		{"blue", emojis.Blue, appouroquest.Blue}, {"teal", emojis.Teal, appouroquest.Teal},
		{"green", emojis.Green, appouroquest.Green}, {"yellow", emojis.Yellow, appouroquest.Yellow},
		{"orange", emojis.Orange, appouroquest.Orange}, {"purple", emojis.Purple, appouroquest.Purple}, {"red", emojis.Red, appouroquest.Red},
	}
	for _, item := range configured {
		if parsed, err := strconv.ParseUint(item.id, 10, 64); err != nil || parsed == 0 {
			return nil, fmt.Errorf("%s emoji ID is invalid", item.name)
		}
		if _, exists := colors[item.id]; exists {
			return nil, fmt.Errorf("duplicate Mudae emoji ID %s", item.id)
		}
		colors[item.id] = item.color
	}
	return &Listener{
		mudaeID: mudaeID, emojiColors: colors, messenger: messenger,
		preferences: preferences, messages: messages, logger: logger,
		gameTimeout: defaultGameTimeout, ctx: context.Background(),
		pending: map[string][]pendingCommand{}, games: map[string]*gameSession{},
	}, nil
}

// Subscribe attaches the feature's message handlers to Discord's gateway.
func (l *Listener) Subscribe(ctx context.Context, session *discordgo.Session) {
	l.mu.Lock()
	l.ctx = ctx
	l.mu.Unlock()
	session.AddHandler(l.handleMessageCreate)
	session.AddHandler(l.handleMessageUpdate)
	session.AddHandler(l.handleMessageDelete)
	session.AddHandler(l.handleMessageDeleteBulk)
}

// handleMessageCreate records exact commands or starts a correlated board.
func (l *Listener) handleMessageCreate(_ *discordgo.Session, event *discordgo.MessageCreate) {
	if event == nil || event.Message == nil || event.GuildID == "" || event.Author == nil {
		return
	}
	if event.Author.ID != l.mudaeID {
		if event.Author.Bot || !isOQCommand(event.Content) {
			return
		}
		userID, err := strconv.ParseUint(event.Author.ID, 10, 64)
		if err != nil || userID == 0 {
			return
		}
		l.recordCommand(event.ChannelID, pendingCommand{recorded: time.Now(), userID: userID, messageID: event.ID})
		return
	}
	if isUnavailable(event.Message) {
		l.cancelCommand(event.ChannelID, referencedMessageID(event.Message))
		return
	}
	snapshot, ok := parseBoard(event.Message, l.emojiColors)
	if !ok || snapshot.terminal {
		return
	}
	correlation, ok := l.consumeCommand(event.ChannelID, referencedMessageID(event.Message))
	if !ok {
		return
	}
	enabled, err := l.preferences.AutoMudaeOQ(correlation.userID)
	if err != nil {
		l.logger.Error("read automatic Ouroquest preference", "error", err)
		return
	}
	if enabled {
		l.startGame(event.Message, snapshot)
	}
}

// handleMessageUpdate forwards component changes to an active game.
func (l *Listener) handleMessageUpdate(_ *discordgo.Session, event *discordgo.MessageUpdate) {
	if event == nil || event.Message == nil || len(event.Components) == 0 {
		return
	}
	snapshot, ok := parseBoard(event.Message, l.emojiColors)
	if !ok {
		return
	}
	l.mu.Lock()
	game := l.games[event.ID]
	l.mu.Unlock()
	if game == nil {
		return
	}
	select {
	case game.updates <- snapshot:
	default:
		l.logger.Warn("ouroquest update buffer is full", "message_id", event.ID)
	}
}

// handleMessageDelete stops one deleted source game.
func (l *Listener) handleMessageDelete(_ *discordgo.Session, event *discordgo.MessageDelete) {
	if event != nil && event.Message != nil {
		l.stopGame(event.ID)
	}
}

// handleMessageDeleteBulk stops every deleted source game.
func (l *Listener) handleMessageDeleteBulk(_ *discordgo.Session, event *discordgo.MessageDeleteBulk) {
	if event != nil {
		for _, id := range event.Messages {
			l.stopGame(id)
		}
	}
}

// recordCommand stores one user-owned correlation in its channel.
func (l *Listener) recordCommand(channel string, command pendingCommand) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.prune(channel, time.Now())
	l.pending[channel] = append(l.pending[channel], command)
}

// prune removes expired command correlations from one channel.
func (l *Listener) prune(channel string, now time.Time) {
	commands := l.pending[channel]
	live := commands[:0]
	for _, command := range commands {
		if now.Sub(command.recorded) <= commandCorrelationTTL {
			live = append(live, command)
		}
	}
	if len(live) == 0 {
		delete(l.pending, channel)
	} else {
		l.pending[channel] = live
	}
}

// consumeCommand resolves an exact reply before falling back to the latest command.
func (l *Listener) consumeCommand(channel, reference string) (pendingCommand, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.prune(channel, time.Now())
	commands := l.pending[channel]
	index := -1
	for i := len(commands) - 1; i >= 0; i-- {
		if reference == "" || commands[i].messageID == reference {
			index = i
			break
		}
	}
	if index < 0 {
		return pendingCommand{}, false
	}
	command := commands[index]
	commands = append(commands[:index], commands[index+1:]...)
	if len(commands) == 0 {
		delete(l.pending, channel)
	} else {
		l.pending[channel] = commands
	}
	return command, true
}

// cancelCommand removes a command that produced an unavailable response.
func (l *Listener) cancelCommand(channel, reference string) {
	_, _ = l.consumeCommand(channel, reference)
}

// startGame creates one isolated actor for a uniquely identified message.
func (l *Listener) startGame(message *discordgo.Message, initial boardSnapshot) bool {
	l.mu.Lock()
	if _, exists := l.games[message.ID]; exists {
		l.mu.Unlock()
		return false
	}
	game := &gameSession{
		sourceID: message.ID, channelID: message.ChannelID, guildID: message.GuildID,
		updates: make(chan boardSnapshot, gameUpdateBuffer), stop: make(chan struct{}),
	}
	l.games[message.ID] = game
	ctx := l.ctx
	l.mu.Unlock()
	go l.runGame(ctx, game, initial)
	return true
}

// runGame serializes solving and Discord writes for one source message.
func (l *Listener) runGame(ctx context.Context, game *gameSession, initial boardSnapshot) {
	timer := time.NewTimer(l.gameTimeout)
	defer timer.Stop()
	helperID := ""
	fingerprint := ""
	revealed := -1
	process := func(snapshot boardSnapshot) bool {
		if snapshot.terminal {
			return false
		}
		if snapshot.revealed < revealed || snapshot.fingerprint == fingerprint || (snapshot.revealed == revealed && fingerprint != "") {
			return true
		}
		result, err := appouroquest.Solve(snapshot.board, snapshot.unavailable)
		var components []discordgo.MessageComponent
		if err != nil {
			components = renderStatus(ptbr.OuroQuestInconsistent)
			l.logger.Warn("ouroquest board is inconsistent", "error", err)
		} else {
			components = renderRecommendation(result)
		}
		creating := helperID == ""
		if creating {
			helperID, err = l.messenger.SendReply(game.channelID, game.guildID, game.sourceID, components)
		} else {
			err = l.messenger.EditMessage(game.channelID, helperID, components)
		}
		if err != nil {
			l.logger.Error("ouroquest helper publish failed", "error", err)
			return !creating
		}
		fingerprint = snapshot.fingerprint
		revealed = snapshot.revealed
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		timer.Reset(l.gameTimeout)
		return true
	}
	if !process(initial) {
		l.finishGame(game, helperID)
		return
	}
	for {
		select {
		case snapshot := <-game.updates:
			if !process(snapshot) {
				l.finishGame(game, helperID)
				return
			}
		case <-game.stop:
			l.finishGame(game, helperID)
			return
		case <-timer.C:
			l.finishGame(game, helperID)
			return
		case <-ctx.Done():
			l.finishGame(game, helperID)
			return
		}
	}
}

// stopGame signals an active game actor exactly once.
func (l *Listener) stopGame(id string) {
	l.mu.Lock()
	game := l.games[id]
	l.mu.Unlock()
	if game != nil {
		game.stopOnce.Do(func() { close(game.stop) })
	}
}

// finishGame removes an actor and best-effort deletes its helper reply.
func (l *Listener) finishGame(game *gameSession, helperID string) {
	if helperID != "" {
		if err := l.messenger.DeleteMessage(game.channelID, helperID); err != nil {
			l.logger.Warn("ouroquest helper cleanup failed", "error", err)
		}
	}
	l.mu.Lock()
	if l.games[game.sourceID] == game {
		delete(l.games, game.sourceID)
	}
	l.mu.Unlock()
}

// isOQCommand recognizes only complete supported Ouroquest commands.
func isOQCommand(content string) bool {
	value := strings.ToLower(strings.TrimSpace(content))
	return value == "$oq" || value == "$ouroquest"
}

// isUnavailable recognizes common localized responses that produce no board.
func isUnavailable(message *discordgo.Message) bool {
	text := strings.ToLower(messageText(message))
	return strings.Contains(text, "você não tem $oq") || strings.Contains(text, "you don't have any $oq")
}

// referencedMessageID returns the command ID when Mudae used a reply.
func referencedMessageID(message *discordgo.Message) string {
	if message.MessageReference == nil {
		return ""
	}
	return message.MessageReference.MessageID
}

// messageText collects stable text surfaces used by response classification.
func messageText(message *discordgo.Message) string {
	var result strings.Builder
	result.WriteString(message.Content)
	for _, embed := range message.Embeds {
		result.WriteString(" " + embed.Title + " " + embed.Description)
	}
	return result.String()
}

// parseBoard validates and maps one 5x5 Mudae button grid.
func parseBoard(message *discordgo.Message, colors map[string]appouroquest.Color) (boardSnapshot, bool) {
	buttons, ok := flattenButtons(message.Components)
	if !ok {
		return boardSnapshot{}, false
	}
	var snapshot boardSnapshot
	var fingerprint strings.Builder
	allDisabled := true
	for position, button := range buttons {
		emoji := ""
		if button.Emoji != nil {
			emoji = button.Emoji.ID
		}
		if color, exists := colors[emoji]; exists {
			snapshot.board[position] = color
			snapshot.revealed++
		}
		snapshot.unavailable[position] = button.Disabled
		allDisabled = allDisabled && button.Disabled
		fmt.Fprintf(&fingerprint, "%s:%t|", emoji, button.Disabled)
	}
	_, spent, valid := progress(snapshot.board)
	snapshot.fingerprint = fingerprint.String()
	snapshot.terminal = !valid || spent >= appouroquest.PaidClickLimit || allDisabled
	return snapshot, true
}

// progress counts targets and paid clicks while validating the red reveal phase.
func progress(board appouroquest.Board) (int, int, bool) {
	found, spent, red := 0, 0, 0
	for _, color := range board {
		switch color {
		case appouroquest.Unknown:
		case appouroquest.Purple:
			found++
		case appouroquest.Red:
			found++
			spent++
			red++
		default:
			spent++
		}
	}
	return found, spent, red <= 1 && (red == 0 || found == appouroquest.TargetCount)
}

// flattenButtons validates the legacy five-row Discord layout used by Mudae.
func flattenButtons(components []discordgo.MessageComponent) ([]discordgo.Button, bool) {
	if len(components) != appouroquest.BoardWidth {
		return nil, false
	}
	buttons := make([]discordgo.Button, 0, appouroquest.CellCount)
	for _, component := range components {
		var children []discordgo.MessageComponent
		switch row := component.(type) {
		case discordgo.ActionsRow:
			children = row.Components
		case *discordgo.ActionsRow:
			children = row.Components
		default:
			return nil, false
		}
		if len(children) != appouroquest.BoardWidth {
			return nil, false
		}
		for _, child := range children {
			switch button := child.(type) {
			case discordgo.Button:
				buttons = append(buttons, button)
			case *discordgo.Button:
				buttons = append(buttons, *button)
			default:
				return nil, false
			}
		}
	}
	return buttons, len(buttons) == appouroquest.CellCount
}
