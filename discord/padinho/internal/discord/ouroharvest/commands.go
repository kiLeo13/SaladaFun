package ouroharvest

import (
	"context"
	"fmt"
	"strconv"

	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/locale/ptbr"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/messagecommand"
)

const (
	toggleCommand = "!toggleohhelper"
	helperCommand = "!ohhelper"
)

// Register declares the Ouroharvest message commands.
func Register(registry *messagecommand.Registry, listener *Listener) {
	registry.Command(toggleCommand, listener.handleToggleCommand)
	registry.Command(helperCommand, listener.handleHelperCommand)
}

// handleToggleCommand toggles only automatic $oh assistance.
func (l *Listener) handleToggleCommand(_ context.Context, request *messagecommand.Request) error {
	if len(request.Arguments) != 0 {
		return request.Responder.Reply(ptbr.OuroHarvestToggleUsage)
	}
	userID, err := strconv.ParseUint(string(request.Actor.UserID), 10, 64)
	if err != nil || userID == 0 {
		return fmt.Errorf("invalid message command user ID %q", request.Actor.UserID)
	}
	enabled, err := l.preferences.ToggleAutoMudaeOH(userID)
	if err != nil {
		return err
	}
	response := ptbr.OuroHarvestAutomaticDisabled
	if enabled {
		response = ptbr.OuroHarvestAutomaticEnabled
	}
	return request.Responder.Reply(response)
}

// handleHelperCommand starts assistance on a replied-to verified board.
func (l *Listener) handleHelperCommand(_ context.Context, request *messagecommand.Request) error {
	if len(request.Arguments) != 0 || request.ReplyToID == "" {
		return request.Responder.Reply(ptbr.OuroHarvestManualUsage)
	}
	message, err := l.messages.LoadMessage(string(request.ChannelID), string(request.ReplyToID))
	if err != nil {
		return fmt.Errorf("load replied-to Ouroharvest board: %w", err)
	}
	if message == nil || message.Author == nil || message.Author.ID != l.mudaeID {
		return request.Responder.Reply(ptbr.OuroHarvestManualNotMudae)
	}
	if message.ChannelID == "" {
		message.ChannelID = string(request.ChannelID)
	}
	if message.GuildID == "" {
		message.GuildID = string(request.GuildID)
	}
	if !l.knowsBoard(message.ID) && !isOuroharvestBoard(message) {
		return request.Responder.Reply(ptbr.OuroHarvestManualNotBoard)
	}
	snapshot, ok := parseBoard(message, l.emojiColors)
	if !ok {
		return request.Responder.Reply(ptbr.OuroHarvestManualNotBoard)
	}
	if snapshot.terminal {
		return request.Responder.Reply(ptbr.OuroHarvestManualFinished)
	}
	l.rememberBoard(message.ID)
	if !l.startGame(message, snapshot) {
		return request.Responder.Reply(ptbr.OuroHarvestManualActive)
	}
	return nil
}
