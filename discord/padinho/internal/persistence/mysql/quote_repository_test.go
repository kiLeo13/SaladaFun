package mysql

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestQuoteRepositoryRandomReturnsEnabledQuoteAndAuthor(t *testing.T) {
	database, mock := quoteMockDatabase(t)
	repository := NewQuoteRepository(database)
	mock.ExpectQuery("SELECT .* FROM `quotes` WHERE enabled = .* ORDER BY RAND\\(\\) LIMIT .* ").
		WithArgs(true, 1).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "author_id", "original_quote", "translated_quote", "source_url", "enabled", "created_at", "updated_at",
		}).AddRow(190, 10, "Time you enjoy wasting, was not wasted.", "O tempo não foi desperdiçado.", nil, true, 1, 2))
	mock.ExpectQuery("SELECT .* FROM `quote_authors` WHERE `quote_authors`.`id` = .* ").
		WithArgs(uint64(10)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "discord_user_id", "created_at", "updated_at"}).AddRow(10, "John Lennon", nil, 1, 2))

	quote, err := repository.Random()
	if err != nil || quote == nil || quote.ID != 190 || quote.Author.Name != "John Lennon" || quote.TranslatedQuote == nil {
		t.Fatalf("Random() = %#v, %v", quote, err)
	}
	assertQuoteExpectations(t, mock)
}

func TestQuoteRepositoryRandomReturnsNilForEmptyCatalog(t *testing.T) {
	database, mock := quoteMockDatabase(t)
	repository := NewQuoteRepository(database)
	mock.ExpectQuery("SELECT .* FROM `quotes` WHERE enabled = .* ORDER BY RAND\\(\\) LIMIT .* ").
		WithArgs(true, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	quote, err := repository.Random()
	if err != nil || quote != nil {
		t.Fatalf("Random() = %#v, %v", quote, err)
	}
	assertQuoteExpectations(t, mock)
}

func TestQuoteRepositoryRandomPropagatesDatabaseFailure(t *testing.T) {
	database, mock := quoteMockDatabase(t)
	repository := NewQuoteRepository(database)
	want := errors.New("database unavailable")
	mock.ExpectQuery("SELECT .* FROM `quotes` WHERE enabled = .* ORDER BY RAND\\(\\) LIMIT .* ").
		WithArgs(true, 1).WillReturnError(want)
	if _, err := repository.Random(); !errors.Is(err, want) {
		t.Fatalf("Random() error = %v", err)
	}
	assertQuoteExpectations(t, mock)
}

func TestQuoteEntityTableNames(t *testing.T) {
	if table := (entity.Quote{}).TableName(); table != "quotes" {
		t.Fatalf("Quote table = %q", table)
	}
	if table := (entity.QuoteAuthor{}).TableName(); table != "quote_authors" {
		t.Fatalf("QuoteAuthor table = %q", table)
	}
}

func quoteMockDatabase(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
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

func assertQuoteExpectations(t *testing.T, mock sqlmock.Sqlmock) {
	t.Helper()
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
