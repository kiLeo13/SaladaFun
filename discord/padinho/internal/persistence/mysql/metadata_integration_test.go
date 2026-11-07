package mysql

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/database"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
)

// TestOptionalMetadataAgainstMySQL verifies nullable years and every metadata column across upserts.
func TestOptionalMetadataAgainstMySQL(t *testing.T) {
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
	title := &entity.LetterboxdTitle{
		TMDBID: uint64(time.Now().UnixNano()), MediaType: "movie", OriginalTitle: "Untitled Maze Runner Reboot",
		Status: new("Planned"), OriginalLanguage: new("en"), OriginCountries: []string{"US"},
		ContentRatings: []entity.ContentRating{}, ExternalIDs: map[string]string{"imdb_id": "tt32276864"}, MetadataFetchedAt: 999,
	}
	if err := r.SaveTitle(ctx, title); err != nil {
		t.Fatal(err)
	}
	id, created := title.ID, title.CreatedAt
	if title.ReleaseYear != nil || title.ReleaseDate != nil || title.RuntimeMinutes != nil || title.EpisodeCount != nil || title.EpisodeRuntimeMinutes != nil || title.Status == nil || *title.Status != "Planned" || title.ExternalIDs["imdb_id"] != "tt32276864" {
		t.Fatalf("unknown metadata: %#v", title)
	}
	date := time.Date(2030, 2, 3, 0, 0, 0, 0, time.UTC)
	title.ReleaseYear, title.ReleaseDate = new(2030), &date
	title.RuntimeMinutes, title.EpisodeCount = new(120), new(42)
	title.EpisodeRuntimeMinutes = []int{45, 50}
	title.Status, title.OriginalLanguage = new("Released"), new("pt")
	title.OriginCountries = []string{"US", "BR"}
	title.ExternalIDs = map[string]string{"imdb_id": "tt123", "tvdb_id": "123"}
	if err := r.SaveTitle(ctx, title); err != nil {
		t.Fatal(err)
	}
	if title.ID != id || title.CreatedAt != created || title.ReleaseYear == nil || *title.ReleaseYear != 2030 || title.ReleaseDate == nil || !title.ReleaseDate.Equal(date) || title.RuntimeMinutes == nil || *title.RuntimeMinutes != 120 || title.EpisodeCount == nil || *title.EpisodeCount != 42 || !reflect.DeepEqual(title.EpisodeRuntimeMinutes, []int{45, 50}) || title.Status == nil || *title.Status != "Released" || title.OriginalLanguage == nil || *title.OriginalLanguage != "pt" || !reflect.DeepEqual(title.OriginCountries, []string{"US", "BR"}) || title.ExternalIDs["tvdb_id"] != "123" {
		t.Fatalf("populated metadata: %#v", title)
	}
	// Empty and NULL updates must clear old values instead of retaining stale metadata.
	title.ReleaseYear, title.ReleaseDate, title.RuntimeMinutes = nil, nil, nil
	title.Status, title.OriginalLanguage = nil, nil
	title.EpisodeCount, title.EpisodeRuntimeMinutes = new(0), []int{}
	title.OriginCountries, title.ExternalIDs = []string{}, map[string]string{}
	if err := r.SaveTitle(ctx, title); err != nil {
		t.Fatal(err)
	}
	if title.ReleaseYear != nil || title.ReleaseDate != nil || title.RuntimeMinutes != nil || title.Status != nil || title.OriginalLanguage != nil || title.EpisodeCount == nil || *title.EpisodeCount != 0 || title.EpisodeRuntimeMinutes == nil || len(title.EpisodeRuntimeMinutes) != 0 || title.OriginCountries == nil || len(title.OriginCountries) != 0 || title.ExternalIDs == nil || len(title.ExternalIDs) != 0 {
		t.Fatalf("cleared metadata: %#v", title)
	}
	// Legacy/unfetched collections remain distinguishable from fetched empty collections.
	title.EpisodeCount, title.EpisodeRuntimeMinutes = nil, nil
	title.OriginCountries, title.ExternalIDs = nil, nil
	if err := r.SaveTitle(ctx, title); err != nil {
		t.Fatal(err)
	}
	if title.EpisodeCount != nil || title.EpisodeRuntimeMinutes != nil || title.OriginCountries != nil || title.ExternalIDs != nil {
		t.Fatalf("unfetched metadata: %#v", title)
	}
	if err := r.SaveWatch(ctx, 1, title.ID, uint64(time.Now().UnixNano()), 5, ""); err != nil {
		t.Fatal(err)
	}
	history, err := r.History(ctx, 1, "date", "desc", 1)
	if err != nil || len(history.Entries) != 1 || history.Entries[0].ReleaseYear != nil {
		t.Fatalf("nullable history: %#v %v", history, err)
	}
	search, err := r.CachedSearch(ctx, title.OriginalTitle, 1)
	if err != nil || len(search.Items) != 1 || search.Items[0].ReleaseYear != nil {
		t.Fatalf("nullable search: %#v %v", search, err)
	}
}
