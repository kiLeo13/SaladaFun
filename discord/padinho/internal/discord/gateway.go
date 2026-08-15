package discord

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/bwmarrin/discordgo"
)

// Gateway owns Padinho's Discord session, registration, and interaction adapter.
type Gateway struct {
	session *discordgo.Session
	routes  *Routes
	logger  *slog.Logger
}

// New constructs a Discord gateway without opening a network connection.
func New(token string, routes *Routes, logger *slog.Logger) (*Gateway, error) {
	session, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, fmt.Errorf("create Discord session: %w", err)
	}
	session.Identify.Intents = discordgo.IntentsGuilds
	return &Gateway{session: session, routes: routes, logger: logger}, nil
}

// Run opens Discord, synchronizes global commands, and blocks until cancellation.
func (g *Gateway) Run(ctx context.Context) error {
	handler := &interactionHandler{routes: g.routes, logger: g.logger, ctx: ctx}
	g.session.AddHandler(handler.handle)
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

func applicationID(session *discordgo.Session) string {
	if session.State.User == nil {
		return ""
	}
	return session.State.User.ID
}
