package birthday

import (
	"context"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/discord"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/locale/ptbr"
)

func (h Handler) ChangePage(_ context.Context, request *discord.InteractionRequest) error {
	direction, month, ownerID, err := parsePage(request.Parameters)
	if err != nil {
		return request.Responder.Respond(ephemeralMessage(ptbr.BirthdayInvalidInteraction))
	}
	if string(request.Actor.UserID) != ownerID {
		return request.Responder.Respond(ephemeralMessage(ptbr.BirthdayOnlyOwner))
	}
	if direction == "previous" && month > time.January {
		month--
	} else if direction == "next" && month < time.December {
		month++
	}
	birthdays, err := h.service.Month(month)
	if err != nil {
		return err
	}
	return request.Responder.Respond(pageResponse(
		discordgo.InteractionResponseUpdateMessage,
		month,
		birthdays,
		ownerID,
	))
}

func ephemeralMessage(message string) *discordgo.InteractionResponse {
	return &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:      discordgo.MessageFlagsEphemeral | discordgo.MessageFlagsIsComponentsV2,
			Components: []discordgo.MessageComponent{discordgo.TextDisplay{Content: message}},
		},
	}
}
