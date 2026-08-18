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
	found, err := repository.FindByUserID(birthday.UserID)
	if err != nil || found == nil || found.Name != "Leonardo" {
		t.Fatalf("FindByUserID() = %#v, %v", found, err)
	}
	missing, err := repository.FindByUserID(900000000000000099)
	if err != nil || missing != nil {
		t.Fatalf("missing FindByUserID() = %#v, %v", missing, err)
	}

	newDate := time.Date(1999, 5, 6, 0, 0, 0, 0, time.UTC)
	updates := []struct {
		name   string
		update func() (bool, error)
		check  func(*entity.Birthday) bool
	}{
		{"name", func() (bool, error) { return repository.UpdateName(birthday.UserID, "Leo") }, func(value *entity.Birthday) bool { return value.Name == "Leo" }},
		{"birthday", func() (bool, error) { return repository.UpdateBirthday(birthday.UserID, newDate) }, func(value *entity.Birthday) bool { return value.Birthday.Equal(newDate) }},
		{"timezone", func() (bool, error) { return repository.UpdateTimeZone(birthday.UserID, "America/Manaus") }, func(value *entity.Birthday) bool { return value.TimeZone == "America/Manaus" }},
		{"message", func() (bool, error) { return repository.UpdateMessage(birthday.UserID, "Nova") }, func(value *entity.Birthday) bool { return value.Message == "Nova" }},
	}
	for _, update := range updates {
		updated, updateErr := update.update()
		if updateErr != nil || !updated {
			t.Fatalf("Update%s() = %v, %v", update.name, updated, updateErr)
		}
		current, findErr := repository.FindByUserID(birthday.UserID)
		if findErr != nil || current == nil || current.UserID != birthday.UserID || !update.check(current) {
			t.Fatalf("row after %s update = %#v, %v", update.name, current, findErr)
		}
	}
	updated, err := repository.UpdateMessage(birthday.UserID, "Nova")
	if err != nil || !updated {
		t.Fatalf("unchanged UpdateMessage() = %v, %v", updated, err)
	}
	updated, err = repository.UpdateName(900000000000000099, "Nobody")
	if err != nil || updated {
		t.Fatalf("missing UpdateName() = %v, %v", updated, err)
	}
	march, err := repository.ListByMonth(time.March)
	if err != nil || len(march) != 0 {
		t.Fatalf("ListByMonth() = %#v, %v", march, err)
	}
	may, err := repository.ListByMonth(time.May)
	if err != nil || len(may) != 1 || may[0].Name != "Leo" || may[0].TimeZone != "America/Manaus" || may[0].Message != "Nova" {
		t.Fatalf("updated ListByMonth() = %#v, %v", may, err)
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
