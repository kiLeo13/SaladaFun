package app

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/command"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/commands"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/config"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/database"
	padinhodiscord "github.com/kiLeo13/SaladaFun/discord/padinho/internal/discord"
)

// Run initializes dependencies in failure-safe order and serves Discord until
// the process context is cancelled.
func Run(ctx context.Context, configuration config.Config, logger *slog.Logger) error {
	connection, err := database.Open(
		ctx, database.Settings{
			Host: configuration.Database.Host, Port: configuration.Database.Port,
			Username: configuration.Database.Username, Password: configuration.Database.Password,
			Name: configuration.Database.Name, MaxOpen: configuration.Database.MaxOpen,
			MaxIdle: configuration.Database.MaxIdle, MaxLifetime: configuration.Database.MaxLifetime,
		},
	)
	if err != nil {
		return err
	}
	defer connection.SQL.Close()
	if err := database.Migrate(ctx, connection.SQL, configuration.Database.MigrationsPath); err != nil {
		return err
	}

	registry := command.NewRegistry()
	commands.Register(registry)
	if err := registry.Freeze(); err != nil {
		return fmt.Errorf("freeze command registry: %w", err)
	}
	gateway, err := padinhodiscord.NewGateway(
		configuration.DiscordToken, configuration.DiscordApplication,
		configuration.DiscordGuild, configuration.SyncCommands, registry, logger,
	)
	if err != nil {
		return err
	}
	return gateway.Run(ctx)
}
