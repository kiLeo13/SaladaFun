package ouroquest

import (
	"context"
	"fmt"
	"strconv"

	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/locale/ptbr"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/messagecommand"
)

const (
	toggleCommand = "!toggleoqhelper"
	helperCommand = "!oqhelper"
)

// Register declares the Ouroquest message commands.
func Register(registry *messagecommand.Registry, listener *Listener) {
	registry.Command(toggleCommand, listener.handleToggleCommand)
	registry.Command(helperCommand, listener.handleHelperCommand)
}

// handleToggleCommand toggles only automatic $oq assistance.
func (l *Listener) handleToggleCommand(_ context.Context, request *messagecommand.Request) error {
	if len(request.Arguments) != 0 {
		return request.Responder.Reply(ptbr.OuroQuestToggleUsage)
	}
	userID, err := strconv.ParseUint(string(request.Actor.UserID), 10, 64)
	if err != nil || userID == 0 {
		return fmt.Errorf("invalid message command user ID %q", request.Actor.UserID)
	}
	enabled, err := l.preferences.ToggleAutoMudaeOQ(userID)
	if err != nil {
		return err
	}
	response := ptbr.OuroQuestAutomaticDisabled
	if enabled {
		response = ptbr.OuroQuestAutomaticEnabled
	}
	return request.Responder.Reply(response)
}

// handleHelperCommand starts manual assistance on the replied-to Mudae board.
func (l *Listener) handleHelperCommand(_ context.Context, request *messagecommand.Request) error {
	if len(request.Arguments) != 0 || request.ReplyToID == "" {
		return request.Responder.Reply(ptbr.OuroQuestManualUsage)
	}
	message, err := l.messages.LoadMessage(string(request.ChannelID), string(request.ReplyToID))
	if err != nil {
		return fmt.Errorf("load replied-to Ouroquest board: %w", err)
	}
	if message == nil || message.Author == nil || message.Author.ID != l.mudaeID {
		return request.Responder.Reply(ptbr.OuroQuestManualNotMudae)
	}
	if message.ChannelID == "" {
		message.ChannelID = string(request.ChannelID)
	}
	if message.GuildID == "" {
		message.GuildID = string(request.GuildID)
	}
	snapshot, ok := parseBoard(message, l.emojiColors)
	if !ok {
		return request.Responder.Reply(ptbr.OuroQuestManualNotBoard)
	}
	if snapshot.terminal {
		return request.Responder.Reply(ptbr.OuroQuestManualFinished)
	}
	if !l.startGame(message, snapshot) {
		return request.Responder.Reply(ptbr.OuroQuestManualActive)
	}
	return nil
}
