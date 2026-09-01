package mysql

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestVoiceActivityLogRepositoryCreatesDeliveryRecord(t *testing.T) {
	database, mock := voiceActivityMockDatabase(t)
	oldChannelID, newChannelID := uint64(3), uint64(4)
	log := &entity.VoiceActivityLog{
		GuildID: 1, UserID: 2, OldChannelID: &oldChannelID, NewChannelID: &newChannelID,
		LogStatus: entity.VoiceActivityLogSent, CreatedAt: 5,
	}
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO `voice_activity_logs`").
		WithArgs(uint64(1), uint64(2), &oldChannelID, &newChannelID, entity.VoiceActivityLogSent, int64(5)).
		WillReturnResult(sqlmock.NewResult(9, 1))
	mock.ExpectCommit()
	if err := NewVoiceActivityLogRepository(database).Create(log); err != nil || log.ID != 9 {
		t.Fatalf("Create() = %#v, %v", log, err)
	}
	assertVoiceActivityExpectations(t, mock)
}

func TestVoiceActivityLogRepositoryWrapsDatabaseFailures(t *testing.T) {
	database, mock := voiceActivityMockDatabase(t)
	want := errors.New("database unavailable")
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO `voice_activity_logs`").WillReturnError(want)
	mock.ExpectRollback()
	if err := NewVoiceActivityLogRepository(database).Create(&entity.VoiceActivityLog{GuildID: 1, UserID: 2, LogStatus: entity.VoiceActivityLogFailed, CreatedAt: 3}); !errors.Is(err, want) {
		t.Fatalf("Create() error = %v", err)
	}
	if table := (entity.VoiceActivityLog{}).TableName(); table != "voice_activity_logs" {
		t.Fatalf("TableName() = %q", table)
	}
	assertVoiceActivityExpectations(t, mock)
}

func voiceActivityMockDatabase(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	connection, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = connection.Close() })
	database, err := gorm.Open(gormmysql.New(gormmysql.Config{Conn: connection, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	return database, mock
}

func assertVoiceActivityExpectations(t *testing.T, mock sqlmock.Sqlmock) {
	t.Helper()
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
