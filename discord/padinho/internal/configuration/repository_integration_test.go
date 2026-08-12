package configuration

import (
	"context"
	"errors"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/config"
	padinhodatabase "github.com/kiLeo13/SaladaFun/discord/padinho/internal/database"
)

func TestRepositoryAgainstMySQL(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	database, err := padinhodatabase.Open(ctx, liveDatabaseConfig(t))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	pool, err := database.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pool.Close() })

	transaction := database.WithContext(ctx).Begin()
	if transaction.Error != nil {
		t.Fatalf("begin repository test: %v", transaction.Error)
	}
	t.Cleanup(func() { _ = transaction.Rollback().Error })
	repository := NewRepository(transaction)
	if err := transaction.Exec("DELETE FROM config WHERE name = ?", AppTokenName).Error; err != nil {
		t.Fatalf("delete test configuration: %v", err)
	}
	if _, err := repository.Get(ctx, AppTokenName); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get() missing error = %v", err)
	}
	if err := transaction.Exec("INSERT INTO config (name, value) VALUES (?, ?)", AppTokenName, "token").Error; err != nil {
		t.Fatalf("insert test configuration: %v", err)
	}
	value, err := repository.Get(ctx, AppTokenName)
	if err != nil || value != "token" {
		t.Fatalf("Get() = %q, %v", value, err)
	}
}

func liveDatabaseConfig(t *testing.T) config.DatabaseConfig {
	t.Helper()
	values := map[string]string{
		"host": os.Getenv("TEST_DATABASE_HOST"), "port": os.Getenv("TEST_DATABASE_PORT"),
		"username": os.Getenv("TEST_DATABASE_USERNAME"), "password": os.Getenv("TEST_DATABASE_PASSWORD"),
		"name": os.Getenv("TEST_DATABASE_NAME"),
	}
	for name, value := range values {
		if value == "" {
			t.Skipf("TEST_DATABASE_%s is not set", name)
		}
	}
	port, err := strconv.ParseUint(values["port"], 10, 16)
	if err != nil || port == 0 {
		t.Fatalf("TEST_DATABASE_PORT is invalid: %q", values["port"])
	}
	return config.DatabaseConfig{
		Host: values["host"], Port: uint16(port), Username: values["username"],
		Password: values["password"], Name: values["name"], MaxOpen: 3,
		MaxIdle: 1, MaxLifetime: time.Minute,
	}
}
