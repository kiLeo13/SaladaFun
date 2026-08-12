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
	t.Setenv("LOG_LEVEL", "debug")

	configuration, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if configuration.SyncCommands || configuration.Database.MaxOpen != 7 || configuration.Database.MaxIdle != 3 || configuration.Database.MaxLifetime != 45*time.Minute || configuration.LogLevel != "debug" {
		t.Fatalf("Load() = %#v", configuration)
	}
}

func TestLoadUsesRuntimeDefaults(t *testing.T) {
	setRequiredEnvironment(t)
	configuration, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !configuration.SyncCommands || configuration.LogLevel != defaultLogLevel {
		t.Fatalf("Load() = %#v", configuration)
	}
}

func TestLoadDatabaseDefaults(t *testing.T) {
	setRequiredDatabaseEnvironment(t)
	configuration, err := loadDatabase()
	if err != nil {
		t.Fatal(err)
	}
	if configuration.Host != "mysql.internal" || configuration.Port != 3306 || configuration.Username != "salada" || configuration.Password != "secret" || configuration.Name != "salada" || configuration.MaxOpen != defaultDatabaseMaxOpen || configuration.MaxIdle != defaultDatabaseMaxIdle || configuration.MaxLifetime != defaultDatabaseMaxLife {
		t.Fatalf("loadDatabase() = %#v", configuration)
	}
}

func TestLoadValidation(t *testing.T) {
	tests := map[string]struct {
		configure func(*testing.T)
		want      error
	}{
		"missing host": {func(t *testing.T) {}, ErrDatabaseHostMissing},
		"missing port": {func(t *testing.T) {
			setRequiredEnvironment(t)
			t.Setenv("DATABASE_PORT", "")
		}, ErrDatabasePortMissing},
		"missing username": {func(t *testing.T) {
			setRequiredEnvironment(t)
			t.Setenv("DATABASE_USERNAME", "")
		}, ErrDatabaseUsernameMissing},
		"missing password": {func(t *testing.T) {
			setRequiredEnvironment(t)
			t.Setenv("DATABASE_PASSWORD", "")
		}, ErrDatabasePasswordMissing},
		"missing name": {func(t *testing.T) {
			setRequiredEnvironment(t)
			t.Setenv("DATABASE_NAME", "")
		}, ErrDatabaseNameMissing},
		"invalid port": {func(t *testing.T) {
			setRequiredEnvironment(t)
			t.Setenv("DATABASE_PORT", "70000")
		}, nil},
		"invalid bool":    {func(t *testing.T) { setRequiredEnvironment(t); t.Setenv("DISCORD_SYNC_COMMANDS", "maybe") }, nil},
		"invalid integer": {func(t *testing.T) { setRequiredEnvironment(t); t.Setenv("DATABASE_MAX_OPEN_CONNECTIONS", "0") }, nil},
		"invalid idle connections": {func(t *testing.T) {
			setRequiredEnvironment(t)
			t.Setenv("DATABASE_MAX_IDLE_CONNECTIONS", "0")
		}, nil},
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
	setRequiredDatabaseEnvironment(t)
}

func setRequiredDatabaseEnvironment(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_HOST", "mysql.internal")
	t.Setenv("DATABASE_PORT", "3306")
	t.Setenv("DATABASE_USERNAME", "salada")
	t.Setenv("DATABASE_PASSWORD", "secret")
	t.Setenv("DATABASE_NAME", "salada")
}

func clearEnvironment(t *testing.T) {
	t.Helper()
	for _, name := range []string{
		"DISCORD_SYNC_COMMANDS", "DATABASE_HOST", "DATABASE_PORT",
		"DATABASE_USERNAME", "DATABASE_PASSWORD", "DATABASE_NAME",
		"DATABASE_MAX_OPEN_CONNECTIONS", "DATABASE_MAX_IDLE_CONNECTIONS",
		"DATABASE_CONNECTION_MAX_LIFETIME",
	} {
		t.Setenv(name, "")
	}
}
