package mysql

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/database"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
	"gorm.io/gorm"
)

// TestReviewTextAgainstMySQL verifies cached headings and canonical stored review revisions.
func TestReviewTextAgainstMySQL(t *testing.T) {
	setLiveEnvironment(t)
	db, err := database.Open()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.Close(db) })
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	t.Cleanup(func() { tx.Rollback() })
	repo := NewLetterboxdRepository(tx)
	ctx := context.Background()
	title := &entity.LetterboxdTitle{TMDBID: uint64(time.Now().UnixNano()), MediaType: "movie", OriginalTitle: "Review title", ReleaseYear: new(2020)}
	if err := repo.SaveTitle(ctx, title); err != nil {
		t.Fatal(err)
	}
	cached, err := repo.TitleByID(ctx, title.ID)
	if err != nil || cached.OriginalTitle != "Review title" {
		t.Fatal("cached heading", err)
	}
	if _, err := repo.TitleByID(ctx, 0); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatal("missing title", err)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := repo.TitleByID(cancelled, title.ID); !errors.Is(err, context.Canceled) {
		t.Fatal("cancelled lookup", err)
	}
	user := uint64(time.Now().UnixNano())
	if err := repo.SaveWatch(ctx, user, title.ID, 100, 5, " \r\n  hello\r\n\n world \r "); err != nil {
		t.Fatal(err)
	}
	state, err := repo.Watch(ctx, user, title.ID)
	if err != nil || state.Review == nil || *state.Review != "hello world" {
		t.Fatal("stored normalization", state, err)
	}
	if err := repo.SaveWatch(ctx, user, title.ID, 101, 4, " \r\n\n "); err != nil {
		t.Fatal(err)
	}
	state, err = repo.Watch(ctx, user, title.ID)
	if err != nil || state.Review != nil || state.Rating != 4 {
		t.Fatal("normalized withdrawal", state, err)
	}
}
