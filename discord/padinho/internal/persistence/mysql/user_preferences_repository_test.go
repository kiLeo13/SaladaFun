package mysql

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestUserPreferencesRepositoryFindsMissingAndStoredRows(t *testing.T) {
	database, mock := preferenceMockDatabase(t)
	repository := NewUserPreferencesRepository(database)
	query := "SELECT .*users_preferences.* WHERE user_id = .* LIMIT .*"
	mock.ExpectQuery(query).WithArgs(uint64(1), 1).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "auto_mudae_oc", "created_at", "updated_at"}).AddRow(1, false, 10, 20))
	preferences, err := repository.FindUserPreferences(1)
	if err != nil || preferences == nil || preferences.AutoMudaeOC == nil || *preferences.AutoMudaeOC {
		t.Fatalf("FindUserPreferences() = %#v, %v", preferences, err)
	}
	mock.ExpectQuery(query).WithArgs(uint64(2), 1).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "auto_mudae_oc", "created_at", "updated_at"}))
	preferences, err = repository.FindUserPreferences(2)
	if err != nil || preferences != nil {
		t.Fatalf("missing FindUserPreferences() = %#v, %v", preferences, err)
	}
	assertPreferenceExpectations(t, mock)
}

func TestUserPreferencesRepositoryTogglesAtomically(t *testing.T) {
	database, mock := preferenceMockDatabase(t)
	repository := NewUserPreferencesRepository(database)
	mock.ExpectExec("INSERT INTO users_preferences").
		WithArgs(uint64(1), false, sqlmock.AnyArg(), sqlmock.AnyArg(), true).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery("SELECT .*users_preferences").WithArgs(uint64(1), 1).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "auto_mudae_oc", "created_at", "updated_at"}).AddRow(1, false, 10, 20))
	enabled, err := repository.ToggleAutoMudaeOC(1, true)
	if err != nil || enabled {
		t.Fatalf("ToggleAutoMudaeOC() = %t, %v", enabled, err)
	}
	assertPreferenceExpectations(t, mock)
}

func TestUserPreferencesRepositoryWrapsDatabaseFailures(t *testing.T) {
	want := errors.New("database unavailable")
	database, mock := preferenceMockDatabase(t)
	repository := NewUserPreferencesRepository(database)
	mock.ExpectQuery("SELECT .*users_preferences").WithArgs(uint64(1), 1).WillReturnError(want)
	if _, err := repository.FindUserPreferences(1); !errors.Is(err, want) {
		t.Fatalf("FindUserPreferences() error = %v", err)
	}
	mock.ExpectExec("INSERT INTO users_preferences").WithArgs(uint64(1), false, sqlmock.AnyArg(), sqlmock.AnyArg(), true).WillReturnError(want)
	if _, err := repository.ToggleAutoMudaeOC(1, true); !errors.Is(err, want) {
		t.Fatalf("ToggleAutoMudaeOC() error = %v", err)
	}
	assertPreferenceExpectations(t, mock)
}

func TestUserPreferencesRepositoryRejectsMissingToggleResult(t *testing.T) {
	database, mock := preferenceMockDatabase(t)
	repository := NewUserPreferencesRepository(database)
	mock.ExpectExec("INSERT INTO users_preferences").
		WithArgs(uint64(1), false, sqlmock.AnyArg(), sqlmock.AnyArg(), true).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery("SELECT .*users_preferences").WithArgs(uint64(1), 1).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "auto_mudae_oc", "created_at", "updated_at"}))
	if _, err := repository.ToggleAutoMudaeOC(1, true); err == nil {
		t.Fatal("ToggleAutoMudaeOC() error = nil")
	}
	assertPreferenceExpectations(t, mock)
}

func TestUserPreferencesRepositoryRejectsNullToggleResult(t *testing.T) {
	database, mock := preferenceMockDatabase(t)
	repository := NewUserPreferencesRepository(database)
	mock.ExpectExec("INSERT INTO users_preferences").
		WithArgs(uint64(1), false, sqlmock.AnyArg(), sqlmock.AnyArg(), true).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery("SELECT .*users_preferences").WithArgs(uint64(1), 1).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "auto_mudae_oc", "created_at", "updated_at"}).AddRow(1, nil, 10, 20))
	if _, err := repository.ToggleAutoMudaeOC(1, true); err == nil {
		t.Fatal("ToggleAutoMudaeOC() error = nil")
	}
	assertPreferenceExpectations(t, mock)
}

func TestUserPreferencesRepositoryPropagatesToggleReadFailure(t *testing.T) {
	want := errors.New("read unavailable")
	database, mock := preferenceMockDatabase(t)
	repository := NewUserPreferencesRepository(database)
	mock.ExpectExec("INSERT INTO users_preferences").
		WithArgs(uint64(1), false, sqlmock.AnyArg(), sqlmock.AnyArg(), true).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery("SELECT .*users_preferences").WithArgs(uint64(1), 1).WillReturnError(want)
	if _, err := repository.ToggleAutoMudaeOC(1, true); !errors.Is(err, want) {
		t.Fatalf("ToggleAutoMudaeOC() error = %v", err)
	}
	assertPreferenceExpectations(t, mock)
}

func TestUserPreferencesEntityTableName(t *testing.T) {
	if table := (entity.UserPreferences{}).TableName(); table != "users_preferences" {
		t.Fatalf("TableName() = %q", table)
	}
}

func preferenceMockDatabase(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	connection, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = connection.Close() })
	database, err := gorm.Open(gormmysql.New(gormmysql.Config{
		Conn: connection, SkipInitializeWithVersion: true,
	}), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	return database, mock
}

func assertPreferenceExpectations(t *testing.T, mock sqlmock.Sqlmock) {
	t.Helper()
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
