package mysql

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/database"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
)

// TestClassificationsAgainstMySQL verifies nullable JSON, refresh upserts and all title readers.
func TestClassificationsAgainstMySQL(t *testing.T) {
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
	user := uint64(time.Now().UnixNano())
	title := &entity.LetterboxdTitle{MediaType: "movie", TMDBID: uint64(time.Now().UnixNano()), OriginalTitle: "Classification fixture", ReleaseYear: 2020}
	if err := repo.SaveTitle(ctx, title); err != nil {
		t.Fatal(err)
	}
	var unknown bool
	if err := tx.Raw("SELECT content_ratings IS NULL FROM letterboxd_titles WHERE id=?", title.ID).Scan(&unknown).Error; err != nil || !unknown {
		t.Fatal("unfetched must be SQL NULL", err)
	}
	ratings := []entity.ContentRating{{Country: "BR", Certification: "16", ReleaseType: 3}, {Country: "US", Certification: "R", ReleaseType: 4}}
	title.ContentRatings = ratings
	if err := repo.SaveTitle(ctx, title); err != nil || !reflect.DeepEqual(title.ContentRatings, ratings) {
		t.Fatal("upsert", err)
	}
	stored, err := repo.FindTitle(ctx, "movie", title.TMDBID)
	if err != nil || !reflect.DeepEqual(stored.ContentRatings, ratings) {
		t.Fatal("lookup", err)
	}
	if err := repo.AddWishlist(ctx, user, title.ID); err != nil {
		t.Fatal(err)
	}
	wishlist, err := repo.Wishlist(ctx, user)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, item := range wishlist {
		if item.ID == title.ID {
			found = reflect.DeepEqual(item.ContentRatings, ratings)
		}
	}
	if !found {
		t.Fatal("wishlist lost classifications")
	}
	random, err := repo.RandomUnwatched(ctx, user)
	if err != nil || random == nil || !reflect.DeepEqual(random.ContentRatings, ratings) {
		t.Fatal("random title lost classifications", err)
	}
	state, err := repo.Watch(ctx, user, title.ID)
	if err != nil || state == nil || !reflect.DeepEqual(state.Title.ContentRatings, ratings) {
		t.Fatal("watch metadata lost classifications", err)
	}
	cached, err := repo.CachedSearch(ctx, title.OriginalTitle)
	if err != nil || len(cached) != 1 || !reflect.DeepEqual(cached[0].ContentRatings, ratings) {
		t.Fatal("cached search", err)
	}
	title.ContentRatings = []entity.ContentRating{}
	if err := repo.SaveTitle(ctx, title); err != nil || title.ContentRatings == nil || len(title.ContentRatings) != 0 {
		t.Fatal("known empty must survive refresh", err)
	}
	title.ContentRatings = nil
	if err := repo.SaveTitle(ctx, title); err != nil || title.ContentRatings != nil {
		t.Fatal("clear to unfetched", err)
	}
}
