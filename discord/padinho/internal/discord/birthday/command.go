package birthday

import (
	"context"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/command"
)

// Handler owns the cohesive /birthdays command, button, and modal flow.
type Handler struct {
	service Service
}

func (h Handler) List(_ context.Context, request *command.CommandRequest) error {
	birthdays, err := h.service.Month(time.January)
	if err != nil {
		return err
	}
	return request.Responder.Respond(pageResponse(
		discordgo.InteractionResponseChannelMessageWithSource,
		time.January,
		birthdays,
	))
}
