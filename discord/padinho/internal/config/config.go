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
	ErrDiscordTokenMissing     = errors.New("DISCORD_TOKEN is required")
	ErrDatabaseHostMissing     = errors.New("DATABASE_HOST is required")
	ErrDatabasePortMissing     = errors.New("DATABASE_PORT is required")
	ErrDatabaseUsernameMissing = errors.New("DATABASE_USERNAME is required")
	ErrDatabasePasswordMissing = errors.New("DATABASE_PASSWORD is required")
	ErrDatabaseNameMissing     = errors.New("DATABASE_NAME is required")
)

// Config contains Padinho's environment-backed runtime configuration.
type Config struct {
	DiscordToken       string
	DiscordApplication string
	DiscordGuild       string
	SyncCommands       bool
	Database           Database
	LogLevel           string
}

// Database contains configuration shared by the bot and migration command.
type Database struct {
	Host           string
	Port           uint16
	Username       string
	Password       string
	Name           string
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
		Database:           database,
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
		Host:           os.Getenv("DATABASE_HOST"),
		Username:       os.Getenv("DATABASE_USERNAME"),
		Password:       os.Getenv("DATABASE_PASSWORD"),
		Name:           os.Getenv("DATABASE_NAME"),
		MigrationsPath: valueOrDefault("MIGRATIONS_PATH", defaultMigrationsPath),
	}
	if database.Host == "" {
		return Database{}, ErrDatabaseHostMissing
	}
	var err error
	if database.Port, err = portValue("DATABASE_PORT"); err != nil {
		return Database{}, err
	}
	if database.Username == "" {
		return Database{}, ErrDatabaseUsernameMissing
	}
	if database.Password == "" {
		return Database{}, ErrDatabasePasswordMissing
	}
	if database.Name == "" {
		return Database{}, ErrDatabaseNameMissing
	}
	if database.MaxOpen, err = intValue("DATABASE_MAX_OPEN_CONNECTIONS", defaultDatabaseMaxOpen); err != nil {
		return Database{}, err
	}
	if database.MaxIdle, err = intValue("DATABASE_MAX_IDLE_CONNECTIONS", defaultDatabaseMaxIdle); err != nil {
		return Database{}, err
	}
	if database.MaxLifetime, err = durationValue("DATABASE_CONNECTION_MAX_LIFETIME", defaultDatabaseMaxLife); err != nil {
		return Database{}, err
	}
	if database.MaxIdle > database.MaxOpen {
		return Database{}, errors.New("DATABASE_MAX_IDLE_CONNECTIONS cannot exceed DATABASE_MAX_OPEN_CONNECTIONS")
	}
	return database, nil
}

func portValue(name string) (uint16, error) {
	value := os.Getenv(name)
	if value == "" {
		return 0, ErrDatabasePortMissing
	}
	parsed, err := strconv.ParseUint(value, 10, 16)
	if err != nil || parsed == 0 {
		return 0, fmt.Errorf("%s must be an integer between 1 and 65535", name)
	}
	return uint16(parsed), nil
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
