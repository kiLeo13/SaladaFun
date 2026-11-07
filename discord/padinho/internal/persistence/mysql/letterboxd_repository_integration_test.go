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
