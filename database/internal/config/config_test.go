package config

import (
	"errors"
	"testing"
)

func TestLoad(t *testing.T) {
	setRequiredEnvironment(t)
	configuration, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if configuration.Host != "mysql.internal" || configuration.Port != 3306 || configuration.Username != "salada" || configuration.Password != "secret" || configuration.DatabaseName != "salada" || configuration.MigrationsPath != defaultMigrationsPath {
		t.Fatalf("Load() = %#v", configuration)
	}
}

func TestLoadOverridesMigrationsPath(t *testing.T) {
	setRequiredEnvironment(t)
	t.Setenv("MIGRATIONS_PATH", "/migrations")
	configuration, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if configuration.MigrationsPath != "/migrations" {
		t.Fatalf("MigrationsPath = %q", configuration.MigrationsPath)
	}
}

func TestLoadValidation(t *testing.T) {
	tests := map[string]struct {
		name string
		want error
	}{
		"missing host":     {"DATABASE_HOST", ErrDatabaseHostMissing},
		"missing port":     {"DATABASE_PORT", ErrDatabasePortMissing},
		"missing username": {"DATABASE_USERNAME", ErrDatabaseUsernameMissing},
		"missing password": {"DATABASE_PASSWORD", ErrDatabasePasswordMissing},
		"missing name":     {"DATABASE_NAME", ErrDatabaseNameMissing},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			setRequiredEnvironment(t)
			t.Setenv(test.name, "")
			_, err := Load()
			if !errors.Is(err, test.want) {
				t.Fatalf("Load() error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestLoadRejectsInvalidPort(t *testing.T) {
	setRequiredEnvironment(t)
	t.Setenv("DATABASE_PORT", "70000")
	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil")
	}
}

func setRequiredEnvironment(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_HOST", "mysql.internal")
	t.Setenv("DATABASE_PORT", "3306")
	t.Setenv("DATABASE_USERNAME", "salada")
	t.Setenv("DATABASE_PASSWORD", "secret")
	t.Setenv("DATABASE_NAME", "salada")
}
