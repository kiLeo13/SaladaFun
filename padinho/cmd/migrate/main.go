package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/kiLeo13/SaladaFun/padinho/internal/config"
	"github.com/kiLeo13/SaladaFun/padinho/internal/database"
)

func main() {
	configuration, err := config.LoadDatabase()
	if err != nil {
		slog.Error("database configuration failed", "error", err)
		os.Exit(1)
	}
	connection, err := database.Open(
		context.Background(), configuration.DSN, configuration.MaxOpen,
		configuration.MaxIdle, configuration.MaxLifetime,
	)
	if err != nil {
		slog.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer connection.SQL.Close()
	if err := database.Migrate(context.Background(), connection.SQL, configuration.MigrationsPath); err != nil {
		slog.Error("migration failed", "error", err)
		os.Exit(1)
	}
	slog.Info("database migrations complete")
}
