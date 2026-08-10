package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

const (
	defaultLogLevel           = "info"
	defaultMigrationsPath     = "/app/migrations"
	defaultDatabaseMaxOpen    = 5
	defaultDatabaseMaxIdle    = 2
	defaultDatabaseMaxLife    = 30 * time.Minute
	defaultCommandSyncEnabled = true
)

var (
	ErrDiscordTokenMissing = errors.New("DISCORD_TOKEN is required")
	ErrDatabaseDSNMissing  = errors.New("DATABASE_DSN is required")
)

// Config contains Padinho's environment-backed runtime configuration.
type Config struct {
	DiscordToken       string
	DiscordApplication string
	DiscordGuild       string
	SyncCommands       bool
	DatabaseDSN        string
	DatabaseMaxOpen    int
	DatabaseMaxIdle    int
	DatabaseMaxLife    time.Duration
	MigrationsPath     string
	LogLevel           string
}

// Database contains configuration shared by the bot and migration command.
type Database struct {
	DSN            string
	MaxOpen        int
	MaxIdle        int
	MaxLifetime    time.Duration
	MigrationsPath string
}

// Load reads and validates configuration from environment variables.
func Load() (Config, error) {
	database, err := LoadDatabase()
	if err != nil {
		return Config{}, err
	}
	config := Config{
		DiscordToken:       os.Getenv("DISCORD_TOKEN"),
		DiscordApplication: os.Getenv("DISCORD_APPLICATION_ID"),
		DiscordGuild:       os.Getenv("DISCORD_GUILD_ID"),
		DatabaseDSN:        database.DSN,
		DatabaseMaxOpen:    database.MaxOpen,
		DatabaseMaxIdle:    database.MaxIdle,
		DatabaseMaxLife:    database.MaxLifetime,
		MigrationsPath:     database.MigrationsPath,
		LogLevel:           valueOrDefault("LOG_LEVEL", defaultLogLevel),
	}
	if config.SyncCommands, err = boolValue("DISCORD_SYNC_COMMANDS", defaultCommandSyncEnabled); err != nil {
		return Config{}, err
	}
	if config.DiscordToken == "" {
		return Config{}, ErrDiscordTokenMissing
	}
	return config, nil
}

// LoadDatabase reads only database settings, allowing migrations to run
// without Discord credentials.
func LoadDatabase() (Database, error) {
	database := Database{
		DSN:            os.Getenv("DATABASE_DSN"),
		MigrationsPath: valueOrDefault("MIGRATIONS_PATH", defaultMigrationsPath),
	}
	var err error
	if database.MaxOpen, err = intValue("DATABASE_MAX_OPEN_CONNECTIONS", defaultDatabaseMaxOpen); err != nil {
		return Database{}, err
	}
	if database.MaxIdle, err = intValue("DATABASE_MAX_IDLE_CONNECTIONS", defaultDatabaseMaxIdle); err != nil {
		return Database{}, err
	}
	if database.MaxLifetime, err = durationValue("DATABASE_CONNECTION_MAX_LIFETIME", defaultDatabaseMaxLife); err != nil {
		return Database{}, err
	}
	if database.DSN == "" {
		return Database{}, ErrDatabaseDSNMissing
	}
	if database.MaxIdle > database.MaxOpen {
		return Database{}, errors.New("DATABASE_MAX_IDLE_CONNECTIONS cannot exceed DATABASE_MAX_OPEN_CONNECTIONS")
	}
	return database, nil
}

func valueOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func boolValue(name string, fallback bool) (bool, error) {
	value := os.Getenv(name)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("parse %s: %w", name, err)
	}
	return parsed, nil
}

func intValue(name string, fallback int) (int, error) {
	value := os.Getenv(name)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		return 0, fmt.Errorf("%s must be a positive integer", name)
	}
	return parsed, nil
}

func durationValue(name string, fallback time.Duration) (time.Duration, error) {
	value := os.Getenv(name)
	if value == "" {
		return fallback, nil
	}
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("%s must be a positive Go duration", name)
	}
	return parsed, nil
}
