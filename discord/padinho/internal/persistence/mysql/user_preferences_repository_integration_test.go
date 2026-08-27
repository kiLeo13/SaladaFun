package mysql

import (
	"testing"

	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/database"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
)

func TestUserPreferencesRepositoryAgainstMySQL(t *testing.T) {
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
	const userID uint64 = 900000000000000091
	if err := transaction.Where("user_id = ?", userID).Delete(&entity.UserPreferences{}).Error; err != nil {
		t.Fatalf("clear user preferences: %v", err)
	}
	repository := NewUserPreferencesRepository(transaction)
	preferences, err := repository.FindUserPreferences(userID)
	if err != nil || preferences != nil {
		t.Fatalf("missing FindUserPreferences() = %#v, %v", preferences, err)
	}
	enabled, err := repository.ToggleAutoMudaeOC(userID, true)
	if err != nil || enabled {
		t.Fatalf("first ToggleAutoMudaeOC() = %t, %v", enabled, err)
	}
	enabled, err = repository.ToggleAutoMudaeOC(userID, true)
	if err != nil || !enabled {
		t.Fatalf("second ToggleAutoMudaeOC() = %t, %v", enabled, err)
	}
	enabled, err = repository.ToggleAutoMudaeOQ(userID, true)
	if err != nil || enabled {
		t.Fatalf("first ToggleAutoMudaeOQ() = %t, %v", enabled, err)
	}
	enabled, err = repository.ToggleAutoMudaeOH(userID, true)
	if err != nil || enabled {
		t.Fatalf("first ToggleAutoMudaeOH() = %t, %v", enabled, err)
	}
}
