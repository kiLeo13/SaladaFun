package mysql

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/database"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
)

// TestTitleMetadataClearingAgainstMySQL verifies empty arrays and zero values replace old metadata.
func TestTitleMetadataClearingAgainstMySQL(t *testing.T) {
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
	r := NewLetterboxdRepository(tx)
	ctx := context.Background()
	title := &entity.LetterboxdTitle{TMDBID: uint64(time.Now().UnixNano()), MediaType: "tv", OriginalTitle: "Metadata", ReleaseYear: new(2021), Creators: []string{"Creator"}, Genres: []string{"Drama"}, Homepage: "https://example.test", SeasonCount: 3}
	if err := r.SaveTitle(ctx, title); err != nil {
		t.Fatal(err)
	}
	title.Creators = nil
	title.Genres = nil
	title.Homepage = "javascript:bad"
	title.SeasonCount = 0
	if err := r.SaveTitle(ctx, title); err != nil {
		t.Fatal(err)
	}
	stored, err := r.FindTitle(ctx, "tv", title.TMDBID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Creators == nil || len(stored.Creators) != 0 || stored.Genres == nil || len(stored.Genres) != 0 || stored.Homepage != "" || stored.SeasonCount != 0 {
		t.Fatalf("metadata not cleared: %#v", stored)
	}
}

// TestLetterboxdAgainstMySQL exercises constraints, ownership and concurrent revisions.
func TestLetterboxdAgainstMySQL(t *testing.T) {
	setLiveEnvironment(t)
	db, err := database.Open()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.Close(db) })
	ctx := context.Background()
	r := NewLetterboxdRepository(db)
	title := &entity.LetterboxdTitle{MediaType: "movie", TMDBID: uint64(time.Now().UnixNano()), OriginalTitle: "Original", ReleaseYear: new(2020), MetadataFetchedAt: 1}
	if err := r.SaveTitle(ctx, title); err != nil {
		t.Fatal(err)
	}
	originalID := title.ID
	created := title.CreatedAt
	t.Cleanup(func() {
		db.Exec("DELETE FROM letterboxd_reviews WHERE watch_id IN (SELECT id FROM letterboxd_watches WHERE title_id=?)", originalID)
		db.Where("title_id=?", originalID).Delete(&entity.LetterboxdWatch{})
		db.Where("title_id=?", originalID).Delete(&entity.LetterboxdWishlistItem{})
		db.Delete(&entity.LetterboxdTitle{}, originalID)
	})
	title.LocalizedTitle = "Translation"
	title.Creators = []string{" Ross Duffer ", "Matt Duffer", "Ross Duffer", ""}
	title.Genres = []string{"Drama", "Comedy, Drama"}
	title.Homepage = "https://example.test/show"
	title.SeasonCount = 5
	if err := r.SaveTitle(ctx, title); err != nil || title.ID != originalID || title.CreatedAt != created {
		t.Fatalf("upsert %#v %v", title, err)
	}
	if len(title.Creators) != 2 || title.Creators[0] != "Ross Duffer" || len(title.Genres) != 2 || title.Genres[1] != "Comedy, Drama" || title.Homepage != "https://example.test/show" || title.SeasonCount != 5 {
		t.Fatalf("metadata upsert round trip: %#v", title)
	}
	var jsonType string
	if err := db.Raw("SELECT JSON_TYPE(creators) FROM letterboxd_titles WHERE id=?", title.ID).Scan(&jsonType).Error; err != nil || jsonType != "ARRAY" {
		t.Fatal("JSON array", err)
	}
	if err := db.Exec("UPDATE letterboxd_titles SET creators = JSON_OBJECT() WHERE id=?", title.ID).Error; err == nil {
		t.Fatal("object accepted as creators")
	}
	if err := db.Exec("UPDATE letterboxd_titles SET season_count = -1 WHERE id=?", title.ID).Error; err == nil {
		t.Fatal("negative seasons")
	}
	if missing, err := r.FindTitle(ctx, "tv", title.TMDBID); err != nil || missing != nil {
		t.Fatal("namespace")
	}
	for i := 0; i < 2; i++ {
		if err := r.AddWishlist(ctx, 11, title.ID); err != nil {
			t.Fatal(err)
		}
	}
	if titles, err := r.Wishlist(ctx, 11); err != nil || len(titles) != 1 || titles[0].LocalizedTitle != "Translation" || len(titles[0].Creators) != 2 || len(titles[0].Genres) != 2 {
		t.Fatal("wishlist")
	}
	if deleted, err := r.RemoveWishlist(ctx, 12, title.ID); err != nil || deleted {
		t.Fatal("ownership")
	}
	for _, v := range []struct {
		id     uint64
		rating int
		text   string
	}{{100, 3, "First"}, {101, 5, "Edited"}, {102, 4, ""}, {101, 2, "Old"}, {102, 1, "Replay"}} {
		if err := r.SaveWatch(ctx, 11, title.ID, v.id, v.rating, v.text); err != nil {
			t.Fatal(err)
		}
	}
	var watch entity.LetterboxdWatch
	db.Where("user_id=? AND title_id=?", 11, title.ID).Take(&watch)
	if watch.Rating != 4 || watch.MutationID != 102 {
		t.Fatalf("replay %#v", watch)
	}
	latest, err := r.CurrentReview(ctx, watch.ID)
	if err != nil || latest == nil || latest.Review != nil {
		t.Fatal("tombstone")
	}
	var revisions []entity.LetterboxdReview
	db.Where("watch_id=?", watch.ID).Order("id").Find(&revisions)
	if len(revisions) != 3 || *revisions[0].Review != "First" || *revisions[1].Review != "Edited" {
		t.Fatalf("revisions %#v", revisions)
	}
	if err := r.SaveWatch(ctx, 12, title.ID, 103, 2, ""); err != nil {
		t.Fatal(err)
	}
	if err := r.SaveWatch(ctx, 11, title.ID, 104, 6, ""); err == nil {
		t.Fatal("rating")
	}
	if err := db.Model(&watch).Update("rating", 6).Error; err == nil {
		t.Fatal("SQL check")
	}
	if err := r.SaveWatch(ctx, 11, title.ID, 105, 4, "x"); err == nil {
		t.Fatal("review")
	}
	cached, err := r.CachedSearch(ctx, "Translation", 1)
	if err != nil || len(cached.Items) != 1 || cached.Items[0].ID != title.ID || len(cached.Items[0].Creators) != 2 || cached.Items[0].Homepage != title.Homepage {
		t.Fatal("cached search")
	}
	opinions, err := r.Reviews(ctx, title.ID)
	if err != nil || len(opinions) != 2 || opinions[0].Review != nil || opinions[0].Rating != 4 {
		t.Fatalf("current opinions %#v %v", opinions, err)
	}
	originalDate := watch.WatchedAt
	if err := r.SaveWatch(ctx, 11, title.ID, 106, 5, ""); err != nil {
		t.Fatal(err)
	}
	var rerated entity.LetterboxdWatch
	if err := db.Where("user_id=? AND title_id=?", 11, title.ID).Take(&rerated).Error; err != nil || rerated.WatchedAt != originalDate {
		t.Fatal("watch date changed")
	}
	// Reusing another watch's review interaction forces a unique-key error;
	// the newly inserted watch must roll back with the failed revision.
	if err := r.SaveWatch(ctx, 14, title.ID, 100, 5, "Collision"); err == nil {
		t.Fatal("expected revision conflict")
	}
	var rolledBack int64
	db.Model(&entity.LetterboxdWatch{}).Where("user_id=? AND title_id=?", 14, title.ID).Count(&rolledBack)
	if rolledBack != 0 {
		t.Fatal("failed mutation was not rolled back")
	}
	var wg sync.WaitGroup
	failures := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if err := r.SaveWatch(ctx, 13, title.ID, uint64(200+i), 5, fmt.Sprintf("Review %d", i)); err != nil {
				failures <- err
			}
		}(i)
	}
	wg.Wait()
	close(failures)
	for err := range failures {
		t.Fatal(err)
	}
	var concurrent entity.LetterboxdWatch
	db.Where("user_id=? AND title_id=?", 13, title.ID).Take(&concurrent)
	if concurrent.MutationID != 207 {
		t.Fatalf("concurrent %#v", concurrent)
	}
	if deleted, err := r.RemoveWishlist(ctx, 11, title.ID); err != nil || !deleted {
		t.Fatal("remove")
	}
}

// TestActivityQueriesAgainstMySQL verifies random eligibility and stable history pages.
func TestActivityQueriesAgainstMySQL(t *testing.T) {
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
	r := NewLetterboxdRepository(tx)
	ctx := context.Background()
	ids := []uint64{}
	base := uint64(time.Now().UnixNano())
	ratings := []int{2, 5, 1, 5}
	for i := 0; i < 5; i++ {
		title := &entity.LetterboxdTitle{TMDBID: base + uint64(i), MediaType: "movie", OriginalTitle: fmt.Sprintf("History %d", i), ReleaseYear: new(2020)}
		if err := r.SaveTitle(ctx, title); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, title.ID)
		if err := r.AddWishlist(ctx, 31, title.ID); err != nil {
			t.Fatal(err)
		}
		if i < 4 {
			if err := r.SaveWatch(ctx, 31, title.ID, base+uint64(i), ratings[i], "Original review"); err != nil {
				t.Fatal(err)
			}
		}
	}
	for i := 0; i < 10; i++ {
		title, err := r.RandomUnwatched(ctx, 31)
		if err != nil || title == nil || title.ID != ids[4] {
			t.Fatal("eligible pool")
		}
	}
	if title, err := r.RandomUnwatched(ctx, 32); err != nil || title != nil {
		t.Fatal("other wishlist")
	}
	if err := r.AddWishlist(ctx, 32, ids[0]); err != nil {
		t.Fatal(err)
	}
	if title, err := r.RandomUnwatched(ctx, 32); err != nil || title.ID != ids[0] {
		t.Fatal("another member watch excluded candidate")
	}
	if err := r.SaveWatch(ctx, 31, ids[0], base+10, 2, ""); err != nil {
		t.Fatal(err)
	}
	state, err := r.Watch(ctx, 31, ids[0])
	if err != nil || state.Rating != 2 || state.Review != nil || state.WatchedAt == 0 {
		t.Fatal("withdrawal state")
	}
	fresh, err := r.Watch(ctx, 32, ids[0])
	if err != nil || fresh.Rating != 0 || fresh.Review != nil {
		t.Fatal("individual state")
	}
	for _, sort := range []string{"date", "rating"} {
		for _, order := range []string{"asc", "desc"} {
			first, err := r.History(ctx, 31, sort, order, 1)
			if err != nil || len(first.Entries) != 3 || first.Total != 4 {
				t.Fatalf("first %v %v", first, err)
			}
			second, err := r.History(ctx, 31, sort, order, 2)
			if err != nil || len(second.Entries) != 1 || second.Total != 4 || first.Entries[2].ID == second.Entries[0].ID {
				t.Fatal("pagination boundary")
			}
			if sort == "rating" && ((order == "asc" && first.Entries[0].Rating != 1) || (order == "desc" && first.Entries[0].Rating != 5)) {
				t.Fatal("rating order")
			}
		}
	}
	if entries, err := r.History(ctx, 32, "date", "desc", 1); err != nil || len(entries.Entries) != 0 || entries.Total != 0 {
		t.Fatal("history ownership")
	}
	if err := r.SaveWatch(ctx, 31, ids[4], base+11, 5, ""); err != nil {
		t.Fatal(err)
	}
	if title, err := r.RandomUnwatched(ctx, 31); err != nil || title != nil {
		t.Fatal("all watched")
	}
}
