package letterboxd

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
)

// TestClassificationCacheRefresh distinguishes unfetched NULL from fetched empty arrays.
func TestClassificationCacheRefresh(t *testing.T) {
	now := time.Now()
	m := &memory{title: &entity.LetterboxdTitle{MediaType: "movie", TMDBID: 1, MetadataFetchedAt: now.UnixMilli()}}
	p := &provider{}
	s := New(m, p, nil)
	s.now = func() time.Time { return now }
	if title, stale, err := s.Title(context.Background(), "movie", 1); err != nil || stale || title.ContentRatings == nil || p.calls != 1 {
		t.Fatal("unfetched ratings were not refreshed", err)
	}
	if _, _, err := s.Title(context.Background(), "movie", 1); err != nil || p.calls != 1 {
		t.Fatal("empty ratings should be cached", err)
	}
	m.title.ContentRatings = nil
	p.err = errors.New("offline")
	if _, stale, err := s.Title(context.Background(), "movie", 1); err != nil || !stale {
		t.Fatal("legacy cache unavailable during outage", err)
	}
}
