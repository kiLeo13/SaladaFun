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
	defaultDatabaseMaxOpen    = 5
	defaultDatabaseMaxIdle    = 2
	defaultDatabaseMaxLife    = 30 * time.Minute
	defaultCommandSyncEnabled = true
)

var (
	ErrDatabaseHostMissing     = errors.New("DATABASE_HOST is required")
	ErrDatabasePortMissing     = errors.New("DATABASE_PORT is required")
	ErrDatabaseUsernameMissing = errors.New("DATABASE_USERNAME is required")
	ErrDatabasePasswordMissing = errors.New("DATABASE_PASSWORD is required")
	ErrDatabaseNameMissing     = errors.New("DATABASE_NAME is required")
)

// Config contains Padinho's environment-backed runtime configuration.
type Config struct {
	DiscordApplication string
	DiscordGuild       string
	SyncCommands       bool
	Database           DatabaseConfig
	LogLevel           string
}

// DatabaseConfig contains Padinho's typed MySQL connection and pool settings.
type DatabaseConfig struct {
	Host        string
	Port        uint16
	Username    string
	Password    string
	Name        string
	MaxOpen     int
	MaxIdle     int
	MaxLifetime time.Duration
}

// Load reads and validates configuration from environment variables.
func Load() (Config, error) {
	database, err := loadDatabase()
	if err != nil {
		return Config{}, err
	}
	config := Config{
		DiscordApplication: os.Getenv("DISCORD_APPLICATION_ID"),
		DiscordGuild:       os.Getenv("DISCORD_GUILD_ID"),
		Database:           database,
		LogLevel:           valueOrDefault("LOG_LEVEL", defaultLogLevel),
	}
	if config.SyncCommands, err = boolValue("DISCORD_SYNC_COMMANDS", defaultCommandSyncEnabled); err != nil {
		return Config{}, err
	}
	return config, nil
}

func loadDatabase() (DatabaseConfig, error) {
	database := DatabaseConfig{
		Host:     os.Getenv("DATABASE_HOST"),
		Username: os.Getenv("DATABASE_USERNAME"),
		Password: os.Getenv("DATABASE_PASSWORD"),
		Name:     os.Getenv("DATABASE_NAME"),
	}
	if database.Host == "" {
		return DatabaseConfig{}, ErrDatabaseHostMissing
	}
	var err error
	if database.Port, err = portValue("DATABASE_PORT"); err != nil {
		return DatabaseConfig{}, err
	}
	if database.Username == "" {
		return DatabaseConfig{}, ErrDatabaseUsernameMissing
	}
	if database.Password == "" {
		return DatabaseConfig{}, ErrDatabasePasswordMissing
	}
	if database.Name == "" {
		return DatabaseConfig{}, ErrDatabaseNameMissing
	}
	if database.MaxOpen, err = intValue("DATABASE_MAX_OPEN_CONNECTIONS", defaultDatabaseMaxOpen); err != nil {
		return DatabaseConfig{}, err
	}
	if database.MaxIdle, err = intValue("DATABASE_MAX_IDLE_CONNECTIONS", defaultDatabaseMaxIdle); err != nil {
		return DatabaseConfig{}, err
	}
	if database.MaxLifetime, err = durationValue("DATABASE_CONNECTION_MAX_LIFETIME", defaultDatabaseMaxLife); err != nil {
		return DatabaseConfig{}, err
	}
	if database.MaxIdle > database.MaxOpen {
		return DatabaseConfig{}, errors.New("DATABASE_MAX_IDLE_CONNECTIONS cannot exceed DATABASE_MAX_OPEN_CONNECTIONS")
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
