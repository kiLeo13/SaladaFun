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
	title := &entity.LetterboxdTitle{MediaType: "movie", TMDBID: uint64(time.Now().UnixNano()), OriginalTitle: "Original", ReleaseYear: 2020, MetadataFetchedAt: 1}
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
	if err := r.SaveTitle(ctx, title); err != nil || title.ID != originalID || title.CreatedAt != created {
		t.Fatalf("upsert %#v %v", title, err)
	}
	if missing, err := r.FindTitle(ctx, "tv", title.TMDBID); err != nil || missing != nil {
		t.Fatal("namespace")
	}
	for i := 0; i < 2; i++ {
		if err := r.AddWishlist(ctx, 11, title.ID); err != nil {
			t.Fatal(err)
		}
	}
	if titles, err := r.Wishlist(ctx, 11); err != nil || len(titles) != 1 || titles[0].LocalizedTitle != "Translation" {
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
	cached, err := r.CachedSearch(ctx, "Translation")
	if err != nil || len(cached) != 1 || cached[0].ID != title.ID {
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
		title := &entity.LetterboxdTitle{TMDBID: base + uint64(i), MediaType: "movie", OriginalTitle: fmt.Sprintf("History %d", i), ReleaseYear: 2020}
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
			if err != nil || len(first) != 4 {
				t.Fatalf("first %v %v", first, err)
			}
			second, err := r.History(ctx, 31, sort, order, 2)
			if err != nil || len(second) != 1 || first[3].ID != second[0].ID {
				t.Fatal("pagination boundary")
			}
			if sort == "rating" && ((order == "asc" && first[0].Rating != 1) || (order == "desc" && first[0].Rating != 5)) {
				t.Fatal("rating order")
			}
		}
	}
	if entries, err := r.History(ctx, 32, "date", "desc", 1); err != nil || len(entries) != 0 {
		t.Fatal("history ownership")
	}
	if err := r.SaveWatch(ctx, 31, ids[4], base+11, 5, ""); err != nil {
		t.Fatal(err)
	}
	if title, err := r.RandomUnwatched(ctx, 31); err != nil || title != nil {
		t.Fatal("all watched")
	}
}
