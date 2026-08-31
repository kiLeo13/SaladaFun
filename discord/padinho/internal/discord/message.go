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
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/messagecommand"
)

type messageCommandHandler struct {
	routes *Routes
	logger *slog.Logger
	ctx    context.Context
}

// handle maps eligible guild messages into the message-command registry.
func (h *messageCommandHandler) handle(session *discordgo.Session, event *discordgo.MessageCreate) {
	if event == nil || event.Message == nil || event.GuildID == "" || event.Author == nil ||
		event.Author.Bot || event.WebhookID != "" {
		return
	}
	responder := newMessageResponder(session, event.Message)
	request := &messagecommand.Request{
		Actor:   command.Actor{UserID: command.Snowflake(event.Author.ID)},
		GuildID: command.Snowflake(event.GuildID), ChannelID: command.Snowflake(event.ChannelID),
		MessageID: command.Snowflake(event.ID), Content: event.Content,
		Responder: responder, ReceivedAt: time.Now().UTC(),
	}
	if event.Member != nil {
		request.Actor.Permissions = event.Member.Permissions
		request.Actor.RoleIDs = make([]command.Snowflake, len(event.Member.Roles))
		for index, roleID := range event.Member.Roles {
			request.Actor.RoleIDs[index] = command.Snowflake(roleID)
		}
	}
	if event.MessageReference != nil {
		request.ReplyToID = command.Snowflake(event.MessageReference.MessageID)
	}

	handled, err := h.routes.Messages().Dispatch(h.ctx, request)
	if !handled || err == nil || responder.responded() {
		return
	}
	response := ptbr.GenericMessageCommandError
	if rejection, ok := command.AsRejection(err); ok {
		response = rejection.Error()
	} else {
		h.logger.Error("message command execution failed", "message_id", event.ID, "error", err)
	}
	if responseErr := responder.Reply(response); responseErr != nil {
		h.logger.Error("message command error response failed", "message_id", event.ID, "error", responseErr)
	}
}

type messageResponder struct {
	session    *discordgo.Session
	message    *discordgo.Message
	mu         sync.Mutex
	didRespond bool
}

// newMessageResponder binds safe replies to one command message.
func newMessageResponder(session *discordgo.Session, message *discordgo.Message) *messageResponder {
	return &messageResponder{session: session, message: message}
}

// Reply sends one non-mentioning soft reply to the command message.
func (r *messageResponder) Reply(content string) error {
	return r.reply(&discordgo.MessageSend{Content: content, AllowedMentions: noAllowedMentions()})
}

// ReplyWithUserMentions sends one reply that permits only user mentions.
func (r *messageResponder) ReplyWithUserMentions(content string) error {
	return r.reply(&discordgo.MessageSend{
		Content: content,
		AllowedMentions: &discordgo.MessageAllowedMentions{
			Parse: []discordgo.AllowedMentionType{discordgo.AllowedMentionTypeUsers},
		},
	})
}

// ReplyWithComponents sends one non-mentioning Components V2 reply to the command message.
func (r *messageResponder) ReplyWithComponents(components []discordgo.MessageComponent) error {
	return r.reply(&discordgo.MessageSend{
		Components: components, Flags: discordgo.MessageFlagsIsComponentsV2,
		AllowedMentions: noAllowedMentions(),
	})
}

// noAllowedMentions explicitly prevents Discord from parsing any mentions.
func noAllowedMentions() *discordgo.MessageAllowedMentions {
	return &discordgo.MessageAllowedMentions{Parse: []discordgo.AllowedMentionType{}}
}

// reply sends one soft reply to the command message.
func (r *messageResponder) reply(message *discordgo.MessageSend) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.didRespond {
		return errors.New("message command already has a response")
	}
	failIfMissing := false
	message.Reference = &discordgo.MessageReference{
		Type: discordgo.MessageReferenceTypeDefault, MessageID: r.message.ID,
		ChannelID: r.message.ChannelID, GuildID: r.message.GuildID, FailIfNotExists: &failIfMissing,
	}
	_, err := r.session.ChannelMessageSendComplex(r.message.ChannelID, message)
	if err != nil {
		return fmt.Errorf("reply to Discord message command: %w", err)
	}
	r.didRespond = true
	return nil
}

// responded reports whether the command already produced its single reply.
func (r *messageResponder) responded() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.didRespond
}
