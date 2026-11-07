package mysql

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	app "github.com/kiLeo13/SaladaFun/discord/padinho/internal/application/letterboxd"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/database"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
)

// TestCachedSearchTotalsAgainstMySQL verifies absolute counts beyond the old 25-row cap.
func TestCachedSearchTotalsAgainstMySQL(t *testing.T) {
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
	stamp := uint64(time.Now().UnixNano())
	query := fmt.Sprintf("SearchProbe%d", stamp)
	for i := 0; i < 26; i++ {
		title := &entity.LetterboxdTitle{TMDBID: stamp + uint64(i), MediaType: "movie", OriginalTitle: fmt.Sprintf("%s %02d", query, i), LocalizedTitle: query, ReleaseYear: new(2020)}
		if err := repo.SaveTitle(ctx, title); err != nil {
			t.Fatal(err)
		}
	}
	first, err := repo.CachedSearch(ctx, query, 1)
	if err != nil || len(first.Items) != 20 || first.TotalResults != 26 || first.TotalPages != 2 {
		t.Fatalf("first page %#v %v", first, err)
	}
	second, err := repo.CachedSearch(ctx, query, 2)
	if err != nil || len(second.Items) != 6 || second.TotalResults != 26 || second.TotalPages != 2 || second.Items[0].ID <= first.Items[19].ID {
		t.Fatalf("second page %#v %v", second, err)
	}
	empty, err := repo.CachedSearch(ctx, query, 3)
	if err != nil || len(empty.Items) != 0 || empty.TotalResults != 26 {
		t.Fatal("past last page", err)
	}
	none, err := repo.CachedSearch(ctx, query+"missing", 1)
	if err != nil || len(none.Items) != 0 || none.TotalResults != 0 || none.TotalPages != 0 {
		t.Fatal("no matches", err)
	}
	for _, page := range []int{0, 501} {
		if _, err := repo.CachedSearch(ctx, query, page); err != app.ErrInvalid {
			t.Fatal("invalid page", err)
		}
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := repo.CachedSearch(cancelled, query, 1); !errors.Is(err, context.Canceled) {
		t.Fatal("query failure", err)
	}
}
