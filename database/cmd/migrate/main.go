package main

import (
	"log/slog"
	"os"

	"github.com/kiLeo13/SaladaFun/database/internal/migration"
)

func main() {
	db, err := migration.Open()
	if err != nil {
		slog.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	if err := migration.Up(db); err != nil {
		slog.Error("migration failed", "error", err)
		os.Exit(1)
	}
	slog.Info("database migrations complete")
}
