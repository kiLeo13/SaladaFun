package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
)

const defaultMigrationsPath = "/app/migrations"

var (
	ErrDatabaseHostMissing     = errors.New("DATABASE_HOST is required")
	ErrDatabasePortMissing     = errors.New("DATABASE_PORT is required")
	ErrDatabaseUsernameMissing = errors.New("DATABASE_USERNAME is required")
	ErrDatabasePasswordMissing = errors.New("DATABASE_PASSWORD is required")
	ErrDatabaseNameMissing     = errors.New("DATABASE_NAME is required")
)

// Config contains the environment-backed settings required by the migration
// executable. Callers provide individual fields rather than a DSN.
type Config struct {
	Host           string
	Port           uint16
	Username       string
	Password       string
	DatabaseName   string
	MigrationsPath string
}

// Load reads and validates migration settings from environment variables.
func Load() (Config, error) {
	configuration := Config{
		Host:           os.Getenv("DATABASE_HOST"),
		Username:       os.Getenv("DATABASE_USERNAME"),
		Password:       os.Getenv("DATABASE_PASSWORD"),
		DatabaseName:   os.Getenv("DATABASE_NAME"),
		MigrationsPath: valueOrDefault("MIGRATIONS_PATH", defaultMigrationsPath),
	}
	if configuration.Host == "" {
		return Config{}, ErrDatabaseHostMissing
	}
	port, err := databasePort()
	if err != nil {
		return Config{}, err
	}
	configuration.Port = port
	if configuration.Username == "" {
		return Config{}, ErrDatabaseUsernameMissing
	}
	if configuration.Password == "" {
		return Config{}, ErrDatabasePasswordMissing
	}
	if configuration.DatabaseName == "" {
		return Config{}, ErrDatabaseNameMissing
	}
	return configuration, nil
}

func databasePort() (uint16, error) {
	value := os.Getenv("DATABASE_PORT")
	if value == "" {
		return 0, ErrDatabasePortMissing
	}
	parsed, err := strconv.ParseUint(value, 10, 16)
	if err != nil || parsed == 0 {
		return 0, fmt.Errorf("DATABASE_PORT must be an integer between 1 and 65535")
	}
	return uint16(parsed), nil
}

func valueOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
