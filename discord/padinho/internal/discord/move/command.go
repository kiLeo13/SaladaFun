package move

import (
	"context"
	"errors"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/command"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/locale/ptbr"
)

// Handler executes the /move-all interaction.
type Handler struct {
	service Service
}

// MoveAll moves every member currently connected to the origin voice channel.
// It deliberately does not inspect the destination member limit: Discord is
// the authority for each individual move attempt.
func (h Handler) MoveAll(_ context.Context, request *command.CommandRequest) error {
	destinationID, err := request.Options.Snowflake("destination")
	if err != nil {
		return err
	}
	originID, err := h.origin(request)
	if err != nil {
		return err
	}

	if originID == destinationID {
		return command.RejectForbidden(ptbr.MoveAllSameChannel)
	}
	if err := h.validateVoiceChannels(request.GuildID, originID, destinationID); err != nil {
		return err
	}
	members, err := h.service.MembersInVoiceChannel(request.GuildID, originID)
	if err != nil {
		return fmt.Errorf("list origin voice members: %w", err)
	}
	if err := request.Responder.Respond(moveResponse(len(members))); err != nil {
		return err
	}

	var moveErr error
	for _, memberID := range members {
		if err := h.service.MoveMember(request.GuildID, memberID, destinationID); err != nil {
			moveErr = errors.Join(moveErr, fmt.Errorf("move member %s: %w", memberID, err))
		}
	}
	return moveErr
}

func (h Handler) origin(request *command.CommandRequest) (command.Snowflake, error) {
	originID, err := request.Options.Snowflake("origin")
	if err == nil {
		return originID, nil
	}
	if !errors.Is(err, command.ErrOptionMissing) {
		return "", err
	}

	originID, connected, err := h.service.CurrentVoiceChannel(request.GuildID, request.Actor.UserID)
	if err != nil {
		return "", fmt.Errorf("read caller voice state: %w", err)
	}
	if !connected {
		return "", command.RejectForbidden(ptbr.MoveAllOriginRequired)
	}
	return originID, nil
}

func (h Handler) validateVoiceChannels(guildID, originID, destinationID command.Snowflake) error {
	for _, channel := range []struct {
		id      command.Snowflake
		message string
	}{
		{id: originID, message: ptbr.MoveAllInvalidOrigin},
		{id: destinationID, message: ptbr.MoveAllInvalidDestination},
	} {
		voice, err := h.service.IsVoiceChannel(guildID, channel.id)
		if err != nil {
			return fmt.Errorf("validate voice channel %s: %w", channel.id, err)
		}
		if !voice {
			return command.RejectForbidden(channel.message)
		}
	}
	return nil
}

func moveResponse(memberCount int) *discordgo.InteractionResponse {
	return &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral | discordgo.MessageFlagsIsComponentsV2,
			Components: []discordgo.MessageComponent{discordgo.TextDisplay{
				Content: fmt.Sprintf(ptbr.MoveAllStarted, memberCount),
			}},
		},
	}
}
