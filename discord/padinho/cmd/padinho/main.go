package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/command"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/commands"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/config"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/configuration"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/database"
	padinhodiscord "github.com/kiLeo13/SaladaFun/discord/padinho/internal/discord"
)

var errRequiredConfigurationEmpty = errors.New("required configuration value is empty")

type configurationRepository interface {
	Get(context.Context, string) (string, error)
}

func main() {
	configuration, err := config.Load()
	if err != nil {
		slog.Error("configuration failed", "error", err)
		os.Exit(1)
	}
	level := slog.LevelInfo
	if configuration.LogLevel == "debug" {
		level = slog.LevelDebug
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, configuration, logger); err != nil {
		logger.Error("Padinho stopped", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, runtimeConfig config.Config, logger *slog.Logger) error {
	databaseConnection, err := database.Open(ctx, runtimeConfig.Database)
	if err != nil {
		return err
	}
	pool, err := databaseConnection.DB()
	if err != nil {
		return fmt.Errorf("access MySQL connection pool: %w", err)
	}
	defer pool.Close()

	configRepository := configuration.NewRepository(databaseConnection)
	discordToken, err := requiredConfigurationValue(ctx, configRepository, configuration.AppTokenName)
	if err != nil {
		return err
	}

	registry := command.NewRegistry()
	commands.Register(registry)
	if err := registry.Freeze(); err != nil {
		return fmt.Errorf("freeze command registry: %w", err)
	}
	gateway, err := padinhodiscord.NewGateway(
		discordToken, runtimeConfig.DiscordApplication,
		runtimeConfig.DiscordGuild, runtimeConfig.SyncCommands, registry, logger,
	)
	if err != nil {
		return err
	}
	return gateway.Run(ctx)
}

func requiredConfigurationValue(ctx context.Context, repository configurationRepository, name string) (string, error) {
	value, err := repository.Get(ctx, name)
	if err != nil {
		return "", fmt.Errorf("load required configuration %q: %w", name, err)
	}
	if value == "" {
		return "", fmt.Errorf("%w: %s", errRequiredConfigurationEmpty, name)
	}
	return value, nil
}
