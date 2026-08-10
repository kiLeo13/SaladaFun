package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/kiLeo13/SaladaFun/padinho/internal/app"
	"github.com/kiLeo13/SaladaFun/padinho/internal/config"
)

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
	if err := app.Run(ctx, configuration, logger); err != nil {
		logger.Error("Padinho stopped", "error", err)
		os.Exit(1)
	}
}
