package discord

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/kiLeo13/SaladaFun/padinho/internal/command"
)

const genericCommandError = "Something went wrong while running that command."

type interactionHandler struct {
	registry *command.Registry
	logger   *slog.Logger
	ctx      context.Context
}

func (h *interactionHandler) handle(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	if interaction.Type != discordgo.InteractionApplicationCommand {
		return
	}
	request, responder, err := mapRequest(session, interaction)
	if err == nil {
		err = h.registry.Dispatch(h.ctx, request)
	}
	if err == nil || responder.responded() {
		return
	}
	response := genericCommandError
	if rejection, ok := command.AsRejection(err); ok {
		response = rejection.Error()
	} else {
		h.logger.Error("command execution failed", "request_id", interaction.ID, "error", err)
	}
	if responseErr := responder.Respond(context.Background(), command.Response{Content: response, Ephemeral: true}); responseErr != nil {
		h.logger.Error("command error response failed", "request_id", interaction.ID, "error", responseErr)
	}
}

func mapRequest(session *discordgo.Session, interaction *discordgo.InteractionCreate) (*command.CommandRequest, *interactionResponder, error) {
	data := interaction.ApplicationCommandData()
	path := command.CommandPath{Command: data.Name}
	options := data.Options
	if len(options) > 0 && options[0].Type == discordgo.ApplicationCommandOptionSubCommand {
		path.Subcommand = options[0].Name
		options = options[0].Options
	} else if len(options) > 0 && options[0].Type == discordgo.ApplicationCommandOptionSubCommandGroup {
		path.Group = options[0].Name
		if len(options[0].Options) == 0 || options[0].Options[0].Type != discordgo.ApplicationCommandOptionSubCommand {
			return nil, newInteractionResponder(session, interaction), errors.New("Discord subcommand group has no subcommand")
		}
		path.Subcommand = options[0].Options[0].Name
		options = options[0].Options[0].Options
	}
	values := make(map[string]any, len(options))
	for _, option := range options {
		value, err := mapOption(option)
		if err != nil {
			return nil, newInteractionResponder(session, interaction), err
		}
		values[option.Name] = value
	}
	actor := command.Actor{}
	if interaction.Member != nil {
		if interaction.Member.User != nil {
			actor.UserID = command.Snowflake(interaction.Member.User.ID)
		}
		actor.RoleIDs = make([]command.Snowflake, len(interaction.Member.Roles))
		for index, role := range interaction.Member.Roles {
			actor.RoleIDs[index] = command.Snowflake(role)
		}
	} else if interaction.User != nil {
		actor.UserID = command.Snowflake(interaction.User.ID)
	}
	responder := newInteractionResponder(session, interaction)
	return &command.CommandRequest{
		Path: path, Actor: actor, GuildID: command.Snowflake(interaction.GuildID),
		ChannelID: command.Snowflake(interaction.ChannelID), Options: command.NewOptionValues(values),
		Responder: responder, RequestID: interaction.ID, ReceivedAt: time.Now().UTC(),
	}, responder, nil
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

func (r *interactionResponder) Respond(_ context.Context, response command.Response) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.didRespond {
		return errors.New("interaction already has an initial response")
	}
	data := &discordgo.InteractionResponseData{Content: response.Content}
	if response.Ephemeral {
		data.Flags = discordgo.MessageFlagsEphemeral
	}
	if err := r.session.InteractionRespond(r.interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource, Data: data,
	}); err != nil {
		return fmt.Errorf("respond to Discord interaction: %w", err)
	}
	r.didRespond = true
	return nil
}

func (r *interactionResponder) responded() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.didRespond
}
