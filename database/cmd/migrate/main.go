package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/kiLeo13/SaladaFun/database/internal/config"
	"github.com/kiLeo13/SaladaFun/database/internal/migration"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	configuration, err := config.Load()
	if err != nil {
		slog.Error("migration configuration failed", "error", err)
		os.Exit(1)
	}
	database, err := migration.Open(ctx, configuration)
	if err != nil {
		slog.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer database.Close()
	if err := migration.Up(ctx, database, configuration.MigrationsPath); err != nil {
		slog.Error("migration failed", "error", err)
		os.Exit(1)
	}
	slog.Info("database migrations complete")
}
