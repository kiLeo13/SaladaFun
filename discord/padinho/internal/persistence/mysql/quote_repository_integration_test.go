package mysql

import (
	"testing"

	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/database"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
)

func TestQuoteRepositoryAgainstMySQL(t *testing.T) {
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
	author := entity.QuoteAuthor{Name: "Padinho quote repository test author"}
	if err := transaction.Create(&author).Error; err != nil {
		t.Fatalf("create author: %v", err)
	}
	translation := "Citação de teste"
	if err := transaction.Create(&entity.Quote{
		AuthorID: author.ID, Author: author, OriginalQuote: "Test quote",
		TranslatedQuote: &translation, Enabled: true,
	}).Error; err != nil {
		t.Fatalf("create quote: %v", err)
	}
	quote, err := NewQuoteRepository(transaction).Random()
	if err != nil || quote == nil || !quote.Enabled || quote.Author.ID == 0 || quote.Author.Name == "" {
		t.Fatalf("Random() = %#v, %v", quote, err)
	}
}
