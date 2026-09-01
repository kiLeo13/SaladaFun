package mysql

import (
	"testing"

	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/database"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
)

func TestVoiceActivityLogRepositoryAgainstMySQL(t *testing.T) {
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
	channelID := uint64(900000000000000300)
	stored := &entity.VoiceActivityLog{
		GuildID: 900000000000000100, UserID: 900000000000000200, NewChannelID: &channelID,
		LogStatus: entity.VoiceActivityLogSent, CreatedAt: 1,
	}
	if err := NewVoiceActivityLogRepository(transaction).Create(stored); err != nil || stored.ID == 0 {
		t.Fatalf("Create() = %#v, %v", stored, err)
	}
	var loaded entity.VoiceActivityLog
	if err := transaction.First(&loaded, stored.ID).Error; err != nil || loaded.NewChannelID == nil || *loaded.NewChannelID != channelID || loaded.LogStatus != entity.VoiceActivityLogSent {
		t.Fatalf("stored voice activity = %#v, %v", loaded, err)
	}
}
