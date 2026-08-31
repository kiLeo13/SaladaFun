package mysql

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestDiscordAccountLinkRepositoryTreeUsesRecursiveQuery(t *testing.T) {
	database, mock := accountLinkMockDatabase(t)
	mock.ExpectQuery("WITH RECURSIVE").WithArgs(uint64(3)).WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "parent_id"}).
		AddRow(1, 1, nil).AddRow(2, 2, 1).AddRow(3, 3, 2))
	links, err := NewDiscordAccountLinkRepository(database).Tree(3)
	if err != nil || len(links) != 3 || links[0].ID != 1 || links[1].ParentID == nil || *links[2].ParentID != 2 {
		t.Fatalf("Tree() = %#v, %v", links, err)
	}
	assertAccountLinkExpectations(t, mock)
}

func TestDiscordAccountLinkRepositoryTreePropagatesFailureAndMapsTable(t *testing.T) {
	database, mock := accountLinkMockDatabase(t)
	want := errors.New("database unavailable")
	mock.ExpectQuery("WITH RECURSIVE").WithArgs(uint64(1)).WillReturnError(want)
	if _, err := NewDiscordAccountLinkRepository(database).Tree(1); !errors.Is(err, want) {
		t.Fatalf("Tree() error = %v", err)
	}
	if table := (entity.DiscordAccountLink{}).TableName(); table != "discord_account_links" {
		t.Fatalf("DiscordAccountLink table = %q", table)
	}
	assertAccountLinkExpectations(t, mock)
}

func accountLinkMockDatabase(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
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

func assertAccountLinkExpectations(t *testing.T, mock sqlmock.Sqlmock) {
	t.Helper()
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
