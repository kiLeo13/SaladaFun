package ouroharvest

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
	appouroharvest "github.com/kiLeo13/SaladaFun/discord/padinho/internal/application/ouroharvest"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/locale/ptbr"
)

const (
	commandCorrelationTTL = 2 * time.Second
	defaultGameTimeout    = 135 * time.Second
	gameUpdateBuffer      = 16
)

// Messenger owns the helper reply lifecycle.
type Messenger interface {
	SendReply(channelID, guildID, sourceMessageID string, components []discordgo.MessageComponent) (string, error)
	EditMessage(channelID, messageID string, components []discordgo.MessageComponent) error
	DeleteMessage(channelID, messageID string) error
}

// PreferenceService resolves and toggles automatic assistance.
type PreferenceService interface {
	AutoMudaeOH(userID uint64) (bool, error)
	ToggleAutoMudaeOH(userID uint64) (bool, error)
}

// MessageLoader retrieves the current state of a manually selected board.
type MessageLoader interface {
	LoadMessage(channelID, messageID string) (*discordgo.Message, error)
}

// EmojiIDs maps every $oh sphere to its custom Discord emoji ID.
type EmojiIDs struct {
	Covered, Blue, Teal, Green, Yellow, Orange string
	Red, Purple, Dark, Light, White            string
}

type pendingCommand struct {
	recorded  time.Time
	userID    uint64
	messageID string
}

type rememberedBoard struct{ expires time.Time }

type gameSession struct {
	sourceID, channelID, guildID string
	updates                      chan boardSnapshot
	stop                         chan struct{}
	stopOnce                     sync.Once
}

// Listener correlates $oh commands and serializes every live helper session.
type Listener struct {
	mudaeID     string
	emojiColors map[string]appouroharvest.Color
	messenger   Messenger
	preferences PreferenceService
	messages    MessageLoader
	solver      *appouroharvest.Solver
	logger      *slog.Logger
	gameTimeout time.Duration

	mu      sync.Mutex
	ctx     context.Context
	pending map[string][]pendingCommand
	games   map[string]*gameSession
	boards  map[string]rememberedBoard
}

// New constructs a validated Ouroharvest listener.
func New(mudaeID string, emojis EmojiIDs, messenger Messenger, preferences PreferenceService, messages MessageLoader, logger *slog.Logger) (*Listener, error) {
	if parsed, err := strconv.ParseUint(mudaeID, 10, 64); err != nil || parsed == 0 {
		return nil, errors.New("Mudae bot ID is invalid")
	}
	if messenger == nil || preferences == nil || messages == nil || logger == nil {
		return nil, errors.New("ouroharvest dependency is nil")
	}
	colors := make(map[string]appouroharvest.Color, 11)
	configured := []struct {
		name, id string
		color    appouroharvest.Color
	}{
		{"covered", emojis.Covered, appouroharvest.Covered}, {"blue", emojis.Blue, appouroharvest.Blue},
		{"teal", emojis.Teal, appouroharvest.Teal}, {"green", emojis.Green, appouroharvest.Green},
		{"yellow", emojis.Yellow, appouroharvest.Yellow}, {"orange", emojis.Orange, appouroharvest.Orange},
		{"red", emojis.Red, appouroharvest.Red}, {"purple", emojis.Purple, appouroharvest.Purple},
		{"dark", emojis.Dark, appouroharvest.Dark}, {"light", emojis.Light, appouroharvest.Light},
		{"white", emojis.White, appouroharvest.White},
	}
	for _, item := range configured {
		if parsed, err := strconv.ParseUint(item.id, 10, 64); err != nil || parsed == 0 {
			return nil, fmt.Errorf("%s emoji ID is invalid", item.name)
		}
		if _, duplicate := colors[item.id]; duplicate {
			return nil, fmt.Errorf("duplicate Mudae emoji ID %s", item.id)
		}
		colors[item.id] = item.color
	}
	return &Listener{
		mudaeID: mudaeID, emojiColors: colors, messenger: messenger, preferences: preferences,
		messages: messages, solver: appouroharvest.NewSolver(), logger: logger,
		gameTimeout: defaultGameTimeout, ctx: context.Background(), pending: map[string][]pendingCommand{},
		games: map[string]*gameSession{}, boards: map[string]rememberedBoard{},
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

// handleMessageCreate records exact commands or starts one verified board.
func (l *Listener) handleMessageCreate(_ *discordgo.Session, event *discordgo.MessageCreate) {
	if event == nil || event.Message == nil || event.GuildID == "" || event.Author == nil {
		return
	}
	if event.Author.ID != l.mudaeID {
		if event.Author.Bot || !isOHCommand(event.Content) {
			return
		}
		userID, err := strconv.ParseUint(event.Author.ID, 10, 64)
		if err == nil && userID != 0 {
			l.recordCommand(event.ChannelID, pendingCommand{recorded: time.Now(), userID: userID, messageID: event.ID})
		}
		return
	}
	if isUnavailable(event.Message) {
		l.cancelCommand(event.ChannelID, referencedMessageID(event.Message))
		return
	}
	if !isOuroharvestBoard(event.Message) {
		return
	}
	snapshot, ok := parseBoard(event.Message, l.emojiColors)
	if !ok || snapshot.terminal {
		return
	}
	command, ok := l.consumeCommand(event.ChannelID, referencedMessageID(event.Message))
	if !ok {
		return
	}
	l.rememberBoard(event.ID)
	enabled, err := l.preferences.AutoMudaeOH(command.userID)
	if err != nil {
		l.logger.Error("read automatic Ouroharvest preference", "error", err)
		return
	}
	if enabled {
		l.startGame(event.Message, snapshot)
	}
}

// handleMessageUpdate forwards a complete component snapshot to its actor.
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
	if snapshot.terminal {
		delete(l.boards, event.ID)
	}
	l.mu.Unlock()
	if game == nil {
		return
	}
	select {
	case game.updates <- snapshot:
	default:
		l.logger.Warn("ouroharvest update buffer is full", "message_id", event.ID)
	}
}

// handleMessageDelete stops one deleted board.
func (l *Listener) handleMessageDelete(_ *discordgo.Session, event *discordgo.MessageDelete) {
	if event != nil && event.Message != nil {
		l.forgetBoard(event.ID)
		l.stopGame(event.ID)
	}
}

// handleMessageDeleteBulk stops every deleted board.
func (l *Listener) handleMessageDeleteBulk(_ *discordgo.Session, event *discordgo.MessageDeleteBulk) {
	if event == nil {
		return
	}
	for _, id := range event.Messages {
		l.forgetBoard(id)
		l.stopGame(id)
	}
}

// recordCommand stores one short-lived command correlation.
func (l *Listener) recordCommand(channel string, command pendingCommand) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.pruneLocked(channel, time.Now())
	l.pending[channel] = append(l.pending[channel], command)
}

// consumeCommand resolves an exact Mudae reply before the latest channel command.
func (l *Listener) consumeCommand(channel, reference string) (pendingCommand, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.pruneLocked(channel, time.Now())
	commands := l.pending[channel]
	index := -1
	for current := len(commands) - 1; current >= 0; current-- {
		if reference == "" || commands[current].messageID == reference {
			index = current
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

// pruneLocked removes expired command and manual-board records.
func (l *Listener) pruneLocked(channel string, now time.Time) {
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
	for id, board := range l.boards {
		if !now.Before(board.expires) {
			delete(l.boards, id)
		}
	}
}

// rememberBoard preserves verified identity for manual activation after updates.
func (l *Listener) rememberBoard(id string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.pruneLocked("", time.Now())
	l.boards[id] = rememberedBoard{expires: time.Now().Add(l.gameTimeout)}
}

// knowsBoard reports whether a board remains within its manual-help window.
func (l *Listener) knowsBoard(id string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.pruneLocked("", time.Now())
	_, exists := l.boards[id]
	return exists
}

// forgetBoard removes manual activation metadata.
func (l *Listener) forgetBoard(id string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.boards, id)
}

// startGame creates one isolated actor for a board.
func (l *Listener) startGame(message *discordgo.Message, initial boardSnapshot) bool {
	l.mu.Lock()
	if _, exists := l.games[message.ID]; exists {
		l.mu.Unlock()
		return false
	}
	game := &gameSession{sourceID: message.ID, channelID: message.ChannelID, guildID: message.GuildID, updates: make(chan boardSnapshot, gameUpdateBuffer), stop: make(chan struct{})}
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
	helperID, fingerprint := "", ""
	process := func(snapshot boardSnapshot) bool {
		if snapshot.terminal {
			return false
		}
		if snapshot.fingerprint == fingerprint {
			return true
		}
		var components []discordgo.MessageComponent
		if position, purple := snapshot.purplePosition(); purple {
			components = renderPurple(position, snapshot.state.ClicksLeft)
		} else {
			result, err := l.solver.Solve(snapshot.state)
			if err != nil || result.Recommendation == nil {
				components = renderStatus(ptbr.OuroHarvestNoSuggestion)
				if err != nil {
					l.logger.Warn("ouroharvest state is invalid", "message_id", game.sourceID, "error", err)
				}
			} else if position, found := snapshot.positionFor(result.Recommendation.Action); found {
				components = renderRecommendation(position, *result.Recommendation)
			} else {
				components = renderStatus(ptbr.OuroHarvestNoSuggestion)
			}
		}
		creating := helperID == ""
		var err error
		if creating {
			helperID, err = l.messenger.SendReply(game.channelID, game.guildID, game.sourceID, components)
		} else {
			err = l.messenger.EditMessage(game.channelID, helperID, components)
		}
		if err != nil {
			l.logger.Error("ouroharvest helper publish failed", "message_id", game.sourceID, "error", err)
			return !creating
		}
		fingerprint = snapshot.fingerprint
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

// stopGame signals an active actor exactly once.
func (l *Listener) stopGame(id string) {
	l.mu.Lock()
	game := l.games[id]
	l.mu.Unlock()
	if game != nil {
		game.stopOnce.Do(func() { close(game.stop) })
	}
}

// finishGame removes one actor and best-effort deletes its helper.
func (l *Listener) finishGame(game *gameSession, helperID string) {
	if helperID != "" {
		if err := l.messenger.DeleteMessage(game.channelID, helperID); err != nil {
			l.logger.Warn("ouroharvest helper cleanup failed", "error", err)
		}
	}
	l.mu.Lock()
	if l.games[game.sourceID] == game {
		delete(l.games, game.sourceID)
	}
	delete(l.boards, game.sourceID)
	l.mu.Unlock()
}

// isOHCommand recognizes complete supported command tokens only.
func isOHCommand(content string) bool {
	value := strings.ToLower(strings.TrimSpace(content))
	return value == "$oh" || value == "$ouroharvest"
}

// isUnavailable recognizes common localized no-use responses.
func isUnavailable(message *discordgo.Message) bool {
	text := strings.ToLower(messageText(message))
	return strings.Contains(text, "você não tem $oh") || strings.Contains(text, "você não tem nenhum $oh") || strings.Contains(text, "you don't have any $oh")
}

// referencedMessageID returns the command ID when Mudae used a reply.
func referencedMessageID(message *discordgo.Message) string {
	if message.MessageReference == nil {
		return ""
	}
	return message.MessageReference.MessageID
}
