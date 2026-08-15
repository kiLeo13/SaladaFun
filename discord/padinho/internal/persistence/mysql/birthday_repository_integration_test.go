package mysql

import (
	"os"
	"testing"
	"time"

	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/database"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
)

func TestBirthdayRepositoryAgainstMySQL(t *testing.T) {
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
	if err := transaction.Exec("DELETE FROM birthday_announcements").Error; err != nil {
		t.Fatalf("clear birthday announcements: %v", err)
	}
	if err := transaction.Exec("DELETE FROM birthdays").Error; err != nil {
		t.Fatalf("clear birthdays: %v", err)
	}
	repository := NewBirthdayRepository(transaction)
	birthday := &entity.Birthday{
		UserID: 900000000000000001, Name: "Leo",
		Birthday: time.Date(2000, 3, 4, 0, 0, 0, 0, time.UTC),
		TimeZone: "America/Sao_Paulo", Message: "Olá",
	}
	if err := repository.Save(birthday); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	birthday.Name = "Leonardo"
	if err := repository.Save(birthday); err != nil {
		t.Fatalf("Save() update error = %v", err)
	}
	march, err := repository.ListByMonth(time.March)
	if err != nil || len(march) != 1 || march[0].Name != "Leonardo" {
		t.Fatalf("ListByMonth() = %#v, %v", march, err)
	}
	all, err := repository.List()
	if err != nil || len(all) != 1 {
		t.Fatalf("List() = %#v, %v", all, err)
	}
	announcementDate := time.Date(2026, 3, 4, 0, 0, 0, 0, time.UTC)
	announced, err := repository.WasAnnounced(birthday.UserID, announcementDate)
	if err != nil || announced {
		t.Fatalf("WasAnnounced() = %v, %v", announced, err)
	}
	for run := 0; run < 2; run++ {
		if err := repository.MarkAnnounced(birthday.UserID, announcementDate); err != nil {
			t.Fatalf("MarkAnnounced() run %d error = %v", run+1, err)
		}
	}
	announced, err = repository.WasAnnounced(birthday.UserID, announcementDate)
	if err != nil || !announced {
		t.Fatalf("WasAnnounced() = %v, %v", announced, err)
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
