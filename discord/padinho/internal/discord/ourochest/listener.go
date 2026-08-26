package ourochest

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	appourochest "github.com/kiLeo13/SaladaFun/discord/padinho/internal/application/ourochest"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/locale/ptbr"
)

const (
	defaultGameTimeout    = 3 * time.Minute
	commandCorrelationTTL = 15 * time.Second
	gameUpdateBuffer      = appourochest.MaxClicks + 3
)

// Messenger owns the reply lifecycle required by the $oc feature.
type Messenger interface {
	SendReply(channelID, guildID, sourceMessageID, content string) (string, error)
	EditMessage(channelID, messageID, content string) error
	DeleteMessage(channelID, messageID string) error
}

// EmojiIDs maps every semantic $oc color to its configured custom emoji ID.
type EmojiIDs struct {
	Blue   string
	Teal   string
	Green  string
	Yellow string
	Orange string
	Red    string
}

// Listener correlates commands, parses Mudae boards, and owns active helper sessions.
type Listener struct {
	mudaeID     string
	emojiColors map[string]appourochest.Color
	messenger   Messenger
	logger      *slog.Logger
	gameTimeout time.Duration

	mu      sync.Mutex
	ctx     context.Context
	pending map[string][]pendingCommand
	games   map[string]*gameSession
}

// gameSession serializes snapshots and cleanup for one Mudae source message.
type gameSession struct {
	sourceID  string
	channelID string
	guildID   string
	updates   chan boardSnapshot
	stop      chan struct{}
	stopOnce  sync.Once
}

// New constructs a validated Mudae $oc listener.
func New(mudaeID string, emojis EmojiIDs, messenger Messenger, logger *slog.Logger) (*Listener, error) {
	if err := validateSnowflake("Mudae bot", mudaeID); err != nil {
		return nil, err
	}
	if messenger == nil {
		return nil, errors.New("ourochest messenger is nil")
	}
	if logger == nil {
		return nil, errors.New("ourochest logger is nil")
	}

	emojiColors := make(map[string]appourochest.Color, 6)
	configured := []struct {
		name  string
		id    string
		color appourochest.Color
	}{
		{"blue", emojis.Blue, appourochest.Blue}, {"teal", emojis.Teal, appourochest.Teal},
		{"green", emojis.Green, appourochest.Green}, {"yellow", emojis.Yellow, appourochest.Yellow},
		{"orange", emojis.Orange, appourochest.Orange}, {"red", emojis.Red, appourochest.Red},
	}
	for _, emoji := range configured {
		if err := validateSnowflake(emoji.name+" emoji", emoji.id); err != nil {
			return nil, err
		}
		if _, duplicate := emojiColors[emoji.id]; duplicate {
			return nil, fmt.Errorf("duplicate Mudae emoji ID %s", emoji.id)
		}
		emojiColors[emoji.id] = emoji.color
	}
	return &Listener{
		mudaeID: mudaeID, emojiColors: emojiColors, messenger: messenger, logger: logger,
		gameTimeout: defaultGameTimeout, ctx: context.Background(),
		pending: make(map[string][]pendingCommand), games: make(map[string]*gameSession),
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

// handleMessageCreate records exact commands or starts a correlated Mudae board.
func (l *Listener) handleMessageCreate(_ *discordgo.Session, event *discordgo.MessageCreate) {
	if event == nil || event.Message == nil || event.GuildID == "" || event.Author == nil {
		return
	}
	if event.Author.ID != l.mudaeID {
		if event.Author.Bot {
			return
		}
		kind, uses, ok := parseCommand(event.Content)
		if ok {
			l.recordCommand(event.ChannelID, kind, uses)
		}
		return
	}

	snapshot, ok := parseBoard(event.Message, l.emojiColors)
	if !ok {
		return
	}
	kind := classifyBoardMessage(event.Message)
	if kind == gameUnknown {
		correlated, hasCorrelation := l.consumeCommand(event.ChannelID, gameUnknown)
		if !hasCorrelation {
			l.logger.Debug("ambiguous Mudae grid ignored", "message_id", event.ID, "channel_id", event.ChannelID)
			return
		}
		kind = correlated
	} else {
		_, _ = l.consumeCommand(event.ChannelID, kind)
	}
	if kind != gameOC {
		return
	}
	l.startGame(event.Message, snapshot)
}

// handleMessageUpdate forwards component changes only to an identified game.
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
		l.logger.Warn("ourochest update buffer is full", "message_id", event.ID)
	}
}

// handleMessageDelete ends a game when Mudae's source message disappears.
func (l *Listener) handleMessageDelete(_ *discordgo.Session, event *discordgo.MessageDelete) {
	if event == nil || event.Message == nil {
		return
	}
	l.stopGame(event.ID)
}

// handleMessageDeleteBulk ends every game removed by one Discord bulk event.
func (l *Listener) handleMessageDeleteBulk(_ *discordgo.Session, event *discordgo.MessageDeleteBulk) {
	if event == nil {
		return
	}
	for _, messageID := range event.Messages {
		l.stopGame(messageID)
	}
}

// recordCommand stores a bounded, exact command correlation for one channel.
func (l *Listener) recordCommand(channelID string, kind gameKind, uses int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	l.pruneCommandsLocked(channelID, now)
	l.pending[channelID] = append(l.pending[channelID], pendingCommand{kind: kind, remaining: uses, recorded: now})
}

// consumeCommand consumes the oldest live correlation when it matches expected.
func (l *Listener) consumeCommand(channelID string, expected gameKind) (gameKind, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.pruneCommandsLocked(channelID, time.Now())
	commands := l.pending[channelID]
	if len(commands) == 0 {
		return gameUnknown, false
	}
	command := commands[0]
	if expected != gameUnknown && command.kind != expected {
		return gameUnknown, false
	}
	commands[0].remaining--
	if commands[0].remaining == 0 {
		commands = commands[1:]
	}
	if len(commands) == 0 {
		delete(l.pending, channelID)
	} else {
		l.pending[channelID] = commands
	}
	return command.kind, true
}

// pruneCommandsLocked removes correlations too old to identify a Mudae response.
func (l *Listener) pruneCommandsLocked(channelID string, now time.Time) {
	commands := l.pending[channelID]
	firstLive := 0
	for firstLive < len(commands) && now.Sub(commands[firstLive].recorded) > commandCorrelationTTL {
		firstLive++
	}
	if firstLive == len(commands) {
		delete(l.pending, channelID)
		return
	}
	if firstLive > 0 {
		l.pending[channelID] = append([]pendingCommand(nil), commands[firstLive:]...)
	}
}

// startGame creates one isolated actor for a uniquely identified Mudae message.
func (l *Listener) startGame(message *discordgo.Message, initial boardSnapshot) {
	l.mu.Lock()
	if _, exists := l.games[message.ID]; exists {
		l.mu.Unlock()
		return
	}
	game := &gameSession{
		sourceID: message.ID, channelID: message.ChannelID, guildID: message.GuildID,
		updates: make(chan boardSnapshot, gameUpdateBuffer), stop: make(chan struct{}),
	}
	l.games[message.ID] = game
	ctx := l.ctx
	l.mu.Unlock()
	go l.runGame(ctx, game, initial)
}

// runGame serializes solving and Discord writes for one source message.
func (l *Listener) runGame(ctx context.Context, game *gameSession, initial boardSnapshot) {
	timer := time.NewTimer(l.gameTimeout)
	defer timer.Stop()
	helperID := ""
	currentFingerprint := ""
	currentRevealed := -1

	process := func(snapshot boardSnapshot) bool {
		if snapshot.terminal {
			return false
		}
		if snapshot.revealed < currentRevealed || snapshot.fingerprint == currentFingerprint {
			return true
		}
		if snapshot.revealed == currentRevealed && currentFingerprint != "" {
			return true
		}
		result, err := appourochest.Solve(snapshot.board, snapshot.unavailable)
		content := ""
		if err != nil {
			content = ptbr.OuroChestInconsistent
			l.logger.Warn("ourochest board is inconsistent", "message_id", game.sourceID, "error", err)
		} else {
			content = renderRecommendations(result)
		}
		creating := helperID == ""
		if creating {
			helperID, err = l.messenger.SendReply(game.channelID, game.guildID, game.sourceID, content)
		} else {
			err = l.messenger.EditMessage(game.channelID, helperID, content)
		}
		if err != nil {
			l.logger.Error("ourochest helper publish failed", "message_id", game.sourceID, "error", err)
			return !creating
		}
		currentFingerprint = snapshot.fingerprint
		currentRevealed = snapshot.revealed
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
func (l *Listener) stopGame(messageID string) {
	l.mu.Lock()
	game := l.games[messageID]
	l.mu.Unlock()
	if game != nil {
		game.stopOnce.Do(func() { close(game.stop) })
	}
}

// finishGame removes the actor and best-effort deletes its helper reply.
func (l *Listener) finishGame(game *gameSession, helperID string) {
	if helperID != "" {
		if err := l.messenger.DeleteMessage(game.channelID, helperID); err != nil {
			l.logger.Warn("ourochest helper cleanup failed", "message_id", game.sourceID, "error", err)
		}
	}
	l.mu.Lock()
	if l.games[game.sourceID] == game {
		delete(l.games, game.sourceID)
	}
	l.mu.Unlock()
}

// validateSnowflake rejects empty, non-numeric, and zero Discord identifiers.
func validateSnowflake(name, value string) error {
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil || parsed == 0 {
		return fmt.Errorf("%s ID is invalid", name)
	}
	return nil
}
