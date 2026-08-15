package discord

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/command"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/locale/ptbr"
)

type interactionHandler struct {
	routes *Routes
	logger *slog.Logger
	ctx    context.Context
}

func (h *interactionHandler) handle(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	responder := newInteractionResponder(session, interaction)
	handled, err := h.routes.dispatch(h.ctx, interaction, responder)
	if !handled {
		return
	}
	if err == nil || responder.responded() {
		return
	}
	response := ptbr.GenericInteractionError
	if rejection, ok := command.AsRejection(err); ok {
		response = rejection.Error()
	} else {
		h.logger.Error("command execution failed", "request_id", interaction.ID, "error", err)
	}
	if responseErr := responder.Respond(ephemeralTextResponse(response)); responseErr != nil {
		h.logger.Error("command error response failed", "request_id", interaction.ID, "error", responseErr)
	}
}

func mapRequest(interaction *discordgo.InteractionCreate, responder command.Responder) (*command.CommandRequest, error) {
	data := interaction.ApplicationCommandData()
	path := command.CommandPath{Command: data.Name}
	options := data.Options
	if len(options) > 0 && options[0].Type == discordgo.ApplicationCommandOptionSubCommand {
		path.Subcommand = options[0].Name
		options = options[0].Options
	} else if len(options) > 0 && options[0].Type == discordgo.ApplicationCommandOptionSubCommandGroup {
		path.Group = options[0].Name
		if len(options[0].Options) == 0 || options[0].Options[0].Type != discordgo.ApplicationCommandOptionSubCommand {
			return nil, errors.New("Discord subcommand group has no subcommand")
		}
		path.Subcommand = options[0].Options[0].Name
		options = options[0].Options[0].Options
	}
	values := make(map[string]any, len(options))
	for _, option := range options {
		value, err := mapOption(option)
		if err != nil {
			return nil, err
		}
		values[option.Name] = value
	}
	return &command.CommandRequest{
		Path: path, Actor: actor(interaction), GuildID: command.Snowflake(interaction.GuildID),
		ChannelID: command.Snowflake(interaction.ChannelID), Options: command.NewOptionValues(values),
		Responder: responder, RequestID: interaction.ID, ReceivedAt: time.Now().UTC(),
	}, nil
}

func mapOption(option *discordgo.ApplicationCommandInteractionDataOption) (any, error) {
	switch option.Type {
	case discordgo.ApplicationCommandOptionString:
		return option.StringValue(), nil
	case discordgo.ApplicationCommandOptionInteger:
		return option.IntValue(), nil
	case discordgo.ApplicationCommandOptionBoolean:
		return option.BoolValue(), nil
	case discordgo.ApplicationCommandOptionUser, discordgo.ApplicationCommandOptionChannel:
		return command.Snowflake(fmt.Sprint(option.Value)), nil
	default:
		return nil, fmt.Errorf("unsupported interaction option type %d", option.Type)
	}
}

type interactionResponder struct {
	session     *discordgo.Session
	interaction *discordgo.InteractionCreate
	mu          sync.Mutex
	didRespond  bool
}

func newInteractionResponder(session *discordgo.Session, interaction *discordgo.InteractionCreate) *interactionResponder {
	return &interactionResponder{session: session, interaction: interaction}
}

func (r *interactionResponder) Respond(response *discordgo.InteractionResponse) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.didRespond {
		return errors.New("interaction already has an initial response")
	}
	if response == nil {
		return errors.New("interaction response is nil")
	}
	if err := r.session.InteractionRespond(r.interaction.Interaction, response); err != nil {
		return fmt.Errorf("respond to Discord interaction: %w", err)
	}
	r.didRespond = true
	return nil
}

func ephemeralTextResponse(message string) *discordgo.InteractionResponse {
	return &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:      discordgo.MessageFlagsEphemeral | discordgo.MessageFlagsIsComponentsV2,
			Components: []discordgo.MessageComponent{discordgo.TextDisplay{Content: message}},
		},
	}
}

func (r *interactionResponder) responded() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.didRespond
}
