package config

import (
	"errors"
	"testing"
	"time"
)

func TestLoadUsesDefaultsAndOverrides(t *testing.T) {
	setRequiredEnvironment(t)
	t.Setenv("DISCORD_SYNC_COMMANDS", "false")
	t.Setenv("DATABASE_MAX_OPEN_CONNECTIONS", "7")
	t.Setenv("DATABASE_MAX_IDLE_CONNECTIONS", "3")
	t.Setenv("DATABASE_CONNECTION_MAX_LIFETIME", "45m")
	t.Setenv("MIGRATIONS_PATH", "/migrations")
	t.Setenv("LOG_LEVEL", "debug")

	configuration, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if configuration.SyncCommands || configuration.DatabaseMaxOpen != 7 || configuration.DatabaseMaxIdle != 3 || configuration.DatabaseMaxLife != 45*time.Minute || configuration.MigrationsPath != "/migrations" || configuration.LogLevel != "debug" {
		t.Fatalf("Load() = %#v", configuration)
	}
}

func TestLoadDatabaseDefaults(t *testing.T) {
	t.Setenv("DATABASE_DSN", "user:pass@tcp(db:3306)/padinho")
	configuration, err := LoadDatabase()
	if err != nil {
		t.Fatal(err)
	}
	if configuration.MaxOpen != defaultDatabaseMaxOpen || configuration.MaxIdle != defaultDatabaseMaxIdle || configuration.MaxLifetime != defaultDatabaseMaxLife || configuration.MigrationsPath != defaultMigrationsPath {
		t.Fatalf("LoadDatabase() = %#v", configuration)
	}
}

func TestLoadValidation(t *testing.T) {
	tests := map[string]struct {
		configure func(*testing.T)
		want      error
	}{
		"missing token":    {func(t *testing.T) { t.Setenv("DATABASE_DSN", "dsn") }, ErrDiscordTokenMissing},
		"missing database": {func(t *testing.T) { t.Setenv("DISCORD_TOKEN", "token") }, ErrDatabaseDSNMissing},
		"invalid bool":     {func(t *testing.T) { setRequiredEnvironment(t); t.Setenv("DISCORD_SYNC_COMMANDS", "maybe") }, nil},
		"invalid integer":  {func(t *testing.T) { setRequiredEnvironment(t); t.Setenv("DATABASE_MAX_OPEN_CONNECTIONS", "0") }, nil},
		"invalid duration": {func(t *testing.T) { setRequiredEnvironment(t); t.Setenv("DATABASE_CONNECTION_MAX_LIFETIME", "soon") }, nil},
		"idle exceeds open": {func(t *testing.T) {
			setRequiredEnvironment(t)
			t.Setenv("DATABASE_MAX_OPEN_CONNECTIONS", "1")
			t.Setenv("DATABASE_MAX_IDLE_CONNECTIONS", "2")
		}, nil},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			clearEnvironment(t)
			test.configure(t)
			_, err := Load()
			if err == nil || test.want != nil && !errors.Is(err, test.want) {
				t.Fatalf("Load() error = %v, want %v", err, test.want)
			}
		})
	}
}

func setRequiredEnvironment(t *testing.T) {
	t.Helper()
	t.Setenv("DISCORD_TOKEN", "token")
	t.Setenv("DATABASE_DSN", "dsn")
}

func clearEnvironment(t *testing.T) {
	t.Helper()
	for _, name := range []string{
		"DISCORD_TOKEN", "DATABASE_DSN", "DISCORD_SYNC_COMMANDS",
		"DATABASE_MAX_OPEN_CONNECTIONS", "DATABASE_MAX_IDLE_CONNECTIONS",
		"DATABASE_CONNECTION_MAX_LIFETIME",
	} {
		t.Setenv(name, "")
	}
}
