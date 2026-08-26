package ourochest

import (
	"context"
	"fmt"
	"strconv"

	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/locale/ptbr"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/messagecommand"
)

const (
	toggleCommand = "!toggleochelper"
	helperCommand = "!ochelper"
)

// Register declares the Ourochest message commands.
func Register(registry *messagecommand.Registry, listener *Listener) {
	registry.Command(toggleCommand, listener.handleToggleCommand)
	registry.Command(helperCommand, listener.handleHelperCommand)
}

// handleToggleCommand toggles only automatic assistance for the invoking user.
func (l *Listener) handleToggleCommand(_ context.Context, request *messagecommand.Request) error {
	if len(request.Arguments) != 0 {
		return request.Responder.Reply(ptbr.OuroChestToggleUsage)
	}
	userID, err := strconv.ParseUint(string(request.Actor.UserID), 10, 64)
	if err != nil || userID == 0 {
		return fmt.Errorf("invalid message command user ID %q", request.Actor.UserID)
	}
	enabled, err := l.preferences.ToggleAutoMudaeOC(userID)
	if err != nil {
		return err
	}
	response := ptbr.OuroChestAutomaticDisabled
	if enabled {
		response = ptbr.OuroChestAutomaticEnabled
	}
	return request.Responder.Reply(response)
}

// handleHelperCommand starts manual assistance on the replied-to Mudae board.
func (l *Listener) handleHelperCommand(_ context.Context, request *messagecommand.Request) error {
	if len(request.Arguments) != 0 || request.ReplyToID == "" {
		return request.Responder.Reply(ptbr.OuroChestManualUsage)
	}
	message, err := l.messages.LoadMessage(string(request.ChannelID), string(request.ReplyToID))
	if err != nil {
		return fmt.Errorf("load replied-to Ourochest board: %w", err)
	}
	if message == nil || message.Author == nil || message.Author.ID != l.mudaeID {
		return request.Responder.Reply(ptbr.OuroChestManualNotMudae)
	}
	if message.ChannelID == "" {
		message.ChannelID = string(request.ChannelID)
	}
	if message.GuildID == "" {
		message.GuildID = string(request.GuildID)
	}
	snapshot, ok := parseBoard(message, l.emojiColors)
	if !ok {
		return request.Responder.Reply(ptbr.OuroChestManualNotBoard)
	}
	kind := l.knownBoard(message.ID)
	if kind == gameUnknown {
		kind = classifyBoardMessage(message)
	}
	if kind != gameOC {
		return request.Responder.Reply(ptbr.OuroChestManualNotOC)
	}
	if snapshot.terminal {
		return request.Responder.Reply(ptbr.OuroChestManualFinished)
	}
	l.rememberBoard(message.ID, gameOC)
	if !l.startGame(message, snapshot) {
		return request.Responder.Reply(ptbr.OuroChestManualActive)
	}
	return nil
}
