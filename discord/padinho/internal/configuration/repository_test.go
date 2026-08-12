package configuration

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const selectValueQuery = "SELECT `value` FROM `config` WHERE name = ? LIMIT ?"

func TestRepositoryGet(t *testing.T) {
	database, mock := mockDatabase(t)
	mock.ExpectQuery(regexp.QuoteMeta(selectValueQuery)).
		WithArgs(AppTokenName, 1).
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow("token"))

	value, err := NewRepository(database).Get(context.Background(), AppTokenName)
	if err != nil || value != "token" {
		t.Fatalf("Get() = %q, %v", value, err)
	}
	assertExpectations(t, mock)
}

func TestRepositoryGetReturnsNotFound(t *testing.T) {
	database, mock := mockDatabase(t)
	mock.ExpectQuery(regexp.QuoteMeta(selectValueQuery)).
		WithArgs(AppTokenName, 1).
		WillReturnRows(sqlmock.NewRows([]string{"value"}))

	_, err := NewRepository(database).Get(context.Background(), AppTokenName)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get() error = %v", err)
	}
	assertExpectations(t, mock)
}

func TestRepositoryGetWrapsDatabaseError(t *testing.T) {
	database, mock := mockDatabase(t)
	want := errors.New("database unavailable")
	mock.ExpectQuery(regexp.QuoteMeta(selectValueQuery)).
		WithArgs(AppTokenName, 1).
		WillReturnError(want)

	_, err := NewRepository(database).Get(context.Background(), AppTokenName)
	if !errors.Is(err, want) {
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
