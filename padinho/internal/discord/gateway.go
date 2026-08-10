package discord

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/bwmarrin/discordgo"
	"github.com/kiLeo13/SaladaFun/padinho/internal/command"
)

// Gateway owns Padinho's Discord session, registration, and interaction adapter.
type Gateway struct {
	session       *discordgo.Session
	registry      *command.Registry
	applicationID string
	guildID       string
	syncCommands  bool
	logger        *slog.Logger
}

// NewGateway constructs a Discord gateway without opening a network connection.
func NewGateway(token, applicationID, guildID string, syncCommands bool, registry *command.Registry, logger *slog.Logger) (*Gateway, error) {
	session, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, fmt.Errorf("create Discord session: %w", err)
	}
	session.Identify.Intents = discordgo.IntentsGuilds
	return &Gateway{
		session: session, registry: registry, applicationID: applicationID,
		guildID: guildID, syncCommands: syncCommands, logger: logger,
	}, nil
}

// Run opens Discord, optionally synchronizes commands, and blocks until cancellation.
func (g *Gateway) Run(ctx context.Context) error {
	handler := &interactionHandler{registry: g.registry, logger: g.logger, ctx: ctx}
	g.session.AddHandler(handler.handle)
	if err := g.session.Open(); err != nil {
		return fmt.Errorf("open Discord gateway: %w", err)
	}
	defer g.session.Close()
	if g.syncCommands {
		if err := g.synchronizeCommands(); err != nil {
			return err
		}
	}
	<-ctx.Done()
	return nil
}

func (g *Gateway) synchronizeCommands() error {
	definitions, err := g.registry.Definitions()
	if err != nil {
		return fmt.Errorf("read command definitions: %w", err)
	}
	commands, err := CompileDefinitions(definitions)
	if err != nil {
		return err
	}
	applicationID := g.applicationID
	if applicationID == "" && g.session.State.User != nil {
		applicationID = g.session.State.User.ID
	}
	if applicationID == "" {
		return errors.New("Discord application ID is unavailable after connecting")
	}
	if _, err := g.session.ApplicationCommandBulkOverwrite(applicationID, g.guildID, commands); err != nil {
		return fmt.Errorf("synchronize Discord application commands: %w", err)
	}
	g.logger.Info("Discord commands synchronized", "count", len(commands), "guild_id", g.guildID)
	return nil
}
