package discord

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/bwmarrin/discordgo"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/command"
)

// Gateway owns Padinho's Discord session, registration, and interaction adapter.
type Gateway struct {
	session     *discordgo.Session
	routes      *Routes
	logger      *slog.Logger
	subscribers []Subscriber
}

// Subscriber registers cohesive feature listeners on Padinho's Discord session.
type Subscriber interface {
	Subscribe(context.Context, *discordgo.Session)
}

// New constructs a Discord gateway without opening a network connection.
func New(token string, routes *Routes, logger *slog.Logger) (*Gateway, error) {
	session, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, fmt.Errorf("create Discord session: %w", err)
	}
	session.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildVoiceStates |
		discordgo.IntentsGuildMessages | discordgo.IntentsMessageContent
	return &Gateway{
		session: session,
		routes:  routes,
		logger:  logger,
	}, nil
}

// Run opens Discord, synchronizes global commands, and blocks until cancellation.
func (g *Gateway) Run(ctx context.Context) error {
	handler := &interactionHandler{
		routes: g.routes,
		logger: g.logger,
		ctx:    ctx,
	}
	g.session.AddHandler(handler.handle)
	messageHandler := &messageCommandHandler{routes: g.routes, logger: g.logger, ctx: ctx}
	g.session.AddHandler(messageHandler.handle)
	for _, subscriber := range g.subscribers {
		subscriber.Subscribe(ctx, g.session)
	}
	if err := g.session.Open(); err != nil {
		return fmt.Errorf("open Discord gateway: %w", err)
	}
	defer g.session.Close()

	if err := g.synchronizeCommands(); err != nil {
		return err
	}
	<-ctx.Done()
	return nil
}

// AddSubscriber registers a feature listener before the gateway is opened.
func (g *Gateway) AddSubscriber(subscriber Subscriber) error {
	if subscriber == nil {
		return errors.New("Discord subscriber is nil")
	}
	g.subscribers = append(g.subscribers, subscriber)
	return nil
}

func (g *Gateway) synchronizeCommands() error {
	definitions, err := g.routes.Commands().Definitions()
	if err != nil {
		return fmt.Errorf("read command definitions: %w", err)
	}
	commands, err := CompileDefinitions(definitions)
	if err != nil {
		return err
	}
	applicationID := applicationID(g.session)
	if applicationID == "" {
		return errors.New("Discord application ID is unavailable after connecting")
	}
	if _, err := g.session.ApplicationCommandBulkOverwrite(applicationID, "", commands); err != nil {
		return fmt.Errorf("synchronize Discord application commands: %w", err)
	}
	g.logger.Info("Discord commands synchronized", "count", len(commands))
	return nil
}

// SendMessage sends a message through Padinho's authenticated Discord session.
func (g *Gateway) SendMessage(channelID string, message *discordgo.MessageSend) error {
	if _, err := g.session.ChannelMessageSendComplex(channelID, message); err != nil {
		return fmt.Errorf("send Discord message: %w", err)
	}
	return nil
}

// SendReply sends a non-mentioning Components V2 reply and returns its message ID.
func (g *Gateway) SendReply(channelID, guildID, sourceMessageID string, components []discordgo.MessageComponent) (string, error) {
	failIfMissing := false
	message, err := g.session.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Components:      components,
		Flags:           discordgo.MessageFlagsIsComponentsV2,
		AllowedMentions: &discordgo.MessageAllowedMentions{},
		Reference: &discordgo.MessageReference{
			Type: discordgo.MessageReferenceTypeDefault, MessageID: sourceMessageID,
			ChannelID: channelID, GuildID: guildID, FailIfNotExists: &failIfMissing,
		},
	})
	if err != nil {
		return "", fmt.Errorf("send Discord reply: %w", err)
	}
	return message.ID, nil
}

// EditMessage replaces the components of one Padinho-owned Discord message.
func (g *Gateway) EditMessage(channelID, messageID string, components []discordgo.MessageComponent) error {
	edit := discordgo.NewMessageEdit(channelID, messageID)
	edit.Components = &components
	edit.Flags = discordgo.MessageFlagsIsComponentsV2
	edit.AllowedMentions = &discordgo.MessageAllowedMentions{}
	if _, err := g.session.ChannelMessageEditComplex(edit); err != nil {
		return fmt.Errorf("edit Discord message: %w", err)
	}
	return nil
}

// DeleteMessage removes one Padinho-owned Discord message.
func (g *Gateway) DeleteMessage(channelID, messageID string) error {
	if err := g.session.ChannelMessageDelete(channelID, messageID); err != nil {
		return fmt.Errorf("delete Discord message: %w", err)
	}
	return nil
}

// LoadMessage retrieves the current Discord payload for a referenced message.
func (g *Gateway) LoadMessage(channelID, messageID string) (*discordgo.Message, error) {
	message, err := g.session.ChannelMessage(channelID, messageID)
	if err != nil {
		return nil, fmt.Errorf("load Discord message: %w", err)
	}
	return message, nil
}

// Guild returns the cached guild data for an interaction's server.
func (g *Gateway) Guild(guildID command.Snowflake) (*discordgo.Guild, error) {
	if guildID == "" {
		return nil, nil
	}
	guild, err := g.session.State.Guild(string(guildID))
	if errors.Is(err, discordgo.ErrStateNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read guild state: %w", err)
	}
	return guild, nil
}

// CurrentVoiceChannel returns the channel that a member currently occupies.
func (g *Gateway) CurrentVoiceChannel(guildID, userID command.Snowflake) (command.Snowflake, bool, error) {
	state, err := g.session.State.VoiceState(string(guildID), string(userID))
	if errors.Is(err, discordgo.ErrStateNotFound) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("read voice state: %w", err)
	}
	return command.Snowflake(state.ChannelID), true, nil
}

// IsVoiceChannel reports whether a cached guild channel accepts voice members.
func (g *Gateway) IsVoiceChannel(guildID, channelID command.Snowflake) (bool, error) {
	channel, err := g.session.State.Channel(string(channelID))
	if errors.Is(err, discordgo.ErrStateNotFound) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read channel: %w", err)
	}
	if channel.GuildID != string(guildID) {
		return false, nil
	}
	return channel.Type == discordgo.ChannelTypeGuildVoice || channel.Type == discordgo.ChannelTypeGuildStageVoice, nil
}

// MembersInVoiceChannel snapshots the members currently connected to a voice channel.
func (g *Gateway) MembersInVoiceChannel(guildID, channelID command.Snowflake) ([]command.Snowflake, error) {
	guild, err := g.session.State.Guild(string(guildID))
	if err != nil {
		return nil, fmt.Errorf("read guild state: %w", err)
	}
	members := make([]command.Snowflake, 0, len(guild.VoiceStates))
	for _, state := range guild.VoiceStates {
		if state.ChannelID == string(channelID) {
			members = append(members, command.Snowflake(state.UserID))
		}
	}
	return members, nil
}

// MoveMember requests that Discord move a member. Channel capacity is not
// pre-checked; Discord evaluates each requested move.
func (g *Gateway) MoveMember(guildID, userID, destinationID command.Snowflake) error {
	destination := string(destinationID)
	if err := g.session.GuildMemberMove(string(guildID), string(userID), &destination); err != nil {
		return fmt.Errorf("move guild member: %w", err)
	}
	return nil
}

func applicationID(session *discordgo.Session) string {
	if session.State.User == nil {
		return ""
	}
	return session.State.User.ID
}
