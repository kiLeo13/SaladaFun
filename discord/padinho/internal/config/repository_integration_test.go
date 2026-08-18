package config

import (
	"errors"
	"os"
	"testing"

	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/database"
)

func TestRepositoryAgainstMySQL(t *testing.T) {
	setLiveEnvironment(t)
	db, err := database.Open()
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { _ = database.Close(db) })
	transaction := db.Begin()
	if transaction.Error != nil {
		t.Fatalf("begin repository test: %v", transaction.Error)
	}
	t.Cleanup(func() { _ = transaction.Rollback().Error })
	repository := New(transaction)
	if err := transaction.Exec("DELETE FROM config WHERE name = ?", AppToken).Error; err != nil {
		t.Fatalf("delete test configuration: %v", err)
	}
	if _, err := repository.Get(AppToken); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get() missing error = %v", err)
	}
	if err := transaction.Exec("INSERT INTO config (name, value) VALUES (?, ?)", AppToken, "token").Error; err != nil {
		t.Fatalf("insert test configuration: %v", err)
	}
	value, err := repository.Get(AppToken)
	if err != nil || value != "token" {
		t.Fatalf("Get() = %q, %v", value, err)
	}
	if err := transaction.Exec("DELETE FROM config WHERE name = ?", BirthdayDefaultMessage).Error; err != nil {
		t.Fatalf("delete default birthday message: %v", err)
	}
	if err := transaction.Exec("INSERT INTO config (name, value) VALUES (?, ?)", BirthdayDefaultMessage, "Feliz aniversário, {mention}!").Error; err != nil {
		t.Fatalf("insert default birthday message: %v", err)
	}
	value, err = repository.BirthdayDefaultMessage()
	if err != nil || value != "Feliz aniversário, {mention}!" {
		t.Fatalf("BirthdayDefaultMessage() = %q, %v", value, err)
	}
}

func setLiveEnvironment(t *testing.T) {
	t.Helper()
	variables := map[string]string{
		"DB_HOST": "TEST_DATABASE_HOST", "DB_PORT": "TEST_DATABASE_PORT",
		"DB_USER": "TEST_DATABASE_USERNAME", "DB_PASSWORD": "TEST_DATABASE_PASSWORD",
		"DB_NAME": "TEST_DATABASE_NAME",
	}
	for target, source := range variables {
		value := os.Getenv(source)
		if value == "" {
			t.Skipf("%s is not set", source)
		}
		t.Setenv(target, value)
	}
}
