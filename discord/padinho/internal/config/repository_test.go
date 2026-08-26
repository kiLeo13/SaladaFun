package config

import (
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const selectValueQuery = "SELECT `value` FROM `config` WHERE name = ? LIMIT ?"

func TestGet(t *testing.T) {
	database, mock := mockDatabase(t)
	mock.ExpectQuery(regexp.QuoteMeta(selectValueQuery)).WithArgs(AppToken, 1).
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow("token"))
	value, err := New(database).Get(AppToken)
	if err != nil || value != "token" {
		t.Fatalf("Get() = %q, %v", value, err)
	}
	assertExpectations(t, mock)
}

func TestGetReturnsNotFound(t *testing.T) {
	database, mock := mockDatabase(t)
	mock.ExpectQuery(regexp.QuoteMeta(selectValueQuery)).WithArgs(AppToken, 1).
		WillReturnRows(sqlmock.NewRows([]string{"value"}))
	if _, err := New(database).Get(AppToken); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get() error = %v", err)
	}
	assertExpectations(t, mock)
}

func TestBirthdayDefaultMessage(t *testing.T) {
	database, mock := mockDatabase(t)
	mock.ExpectQuery(regexp.QuoteMeta(selectValueQuery)).WithArgs(BirthdayDefaultMessage, 1).
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow("Feliz aniversário, {mention}!"))
	value, err := New(database).BirthdayDefaultMessage()
	if err != nil || value != "Feliz aniversário, {mention}!" {
		t.Fatalf("BirthdayDefaultMessage() = %q, %v", value, err)
	}
	assertExpectations(t, mock)
}

func TestMudaeOC(t *testing.T) {
	database, mock := mockDatabase(t)
	want := MudaeOCSettings{
		BotID: "100", BlueEmojiID: "101", TealEmojiID: "102", GreenEmojiID: "103",
		YellowEmojiID: "104", OrangeEmojiID: "105", RedEmojiID: "106",
	}
	values := []struct{ name, value string }{
		{MudaeBotID, want.BotID}, {MudaeOCBlueEmojiID, want.BlueEmojiID},
		{MudaeOCTealEmojiID, want.TealEmojiID}, {MudaeOCGreenEmojiID, want.GreenEmojiID},
		{MudaeOCYellowEmojiID, want.YellowEmojiID}, {MudaeOCOrangeEmojiID, want.OrangeEmojiID},
		{MudaeOCRedEmojiID, want.RedEmojiID},
	}
	for _, value := range values {
		mock.ExpectQuery(regexp.QuoteMeta(selectValueQuery)).WithArgs(value.name, 1).
			WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(value.value))
	}
	got, err := New(database).MudaeOC()
	if err != nil || got != want {
		t.Fatalf("MudaeOC() = %#v, %v", got, err)
	}
	assertExpectations(t, mock)
}

func TestMudaeOCReturnsFirstReadError(t *testing.T) {
	database, mock := mockDatabase(t)
	mock.ExpectQuery(regexp.QuoteMeta(selectValueQuery)).WithArgs(MudaeBotID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"value"}))
	if _, err := New(database).MudaeOC(); !errors.Is(err, ErrNotFound) {
		t.Fatalf("MudaeOC() error = %v", err)
	}
	assertExpectations(t, mock)
}

func TestGetWrapsDatabaseError(t *testing.T) {
	database, mock := mockDatabase(t)
	want := errors.New("database unavailable")
	mock.ExpectQuery(regexp.QuoteMeta(selectValueQuery)).WithArgs(AppToken, 1).WillReturnError(want)
	if _, err := New(database).Get(AppToken); !errors.Is(err, want) {
		t.Fatalf("Get() error = %v", err)
	}
	assertExpectations(t, mock)
}

func TestEntryTableName(t *testing.T) {
	if name := (entry{}).TableName(); name != "config" {
		t.Fatalf("TableName() = %q", name)
	}
}

func mockDatabase(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	connection, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = connection.Close() })
	database, err := gorm.Open(
		gormmysql.New(gormmysql.Config{Conn: connection, SkipInitializeWithVersion: true}),
		&gorm.Config{},
	)
	if err != nil {
		t.Fatal(err)
	}
	return database, mock
}

func assertExpectations(t *testing.T, mock sqlmock.Sqlmock) {
	t.Helper()
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
