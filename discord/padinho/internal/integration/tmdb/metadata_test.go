package tmdb

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	app "github.com/kiLeo13/SaladaFun/discord/padinho/internal/application/letterboxd"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
)

type metadataCache struct {
	title *entity.LetterboxdTitle
	saves int
}

// TestMetadataFailures rejects inconsistent identities and invalid nonempty provider values.
func TestMetadataFailures(t *testing.T) {
	for _, tc := range []struct{ kind, body string }{
		{"movie", `{"id":1,"original_title":"Movie","runtime":-1}`},
		{"tv", `{"id":2,"original_name":"TV"}`},
		{"movie", `{"id":1,"original_title":"Movie","release_date":"abcd"}`},
	} {
		c, _ := New("secret", http.Client{Transport: transport(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(tc.body))}, nil
		})})
		if _, err := c.Details(tc.kind, 1); err == nil {
			t.Fatalf("invalid metadata accepted: %s", tc.body)
		}
	}
	c, _ := New("secret", http.Client{Transport: transport(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"id":1,"title":"Fallback"}`))}, nil
	})})
	if title, err := c.Details("movie", 1); err != nil || title.OriginalTitle != "Fallback" {
		t.Fatalf("fallback title: %#v %v", title, err)
	}
}

// TestOptionalSearchYears preserves unknown years and safe title fallbacks in suggestions.
func TestOptionalSearchYears(t *testing.T) {
	for _, body := range []string{
		`{}`,
		`{"results":[{"id":1,"media_type":"movie","title":"Planned","release_date":""},{"id":2,"media_type":"tv","name":"Series","first_air_date":"abcd"},{"id":3,"media_type":"movie","title":"Zero","release_date":"0000"}]}`,
	} {
		c, _ := New("secret", http.Client{Transport: transport(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
		})})
		result, err := c.Search("planned", 1)
		if err != nil {
			t.Fatal(err)
		}
		for _, title := range result.Items {
			if title.ReleaseYear != nil || title.OriginalTitle == "" {
				t.Fatalf("unknown search year: %#v", title)
			}
		}
	}
	c, _ := New("secret", http.Client{Transport: transport(func(*http.Request) (*http.Response, error) { return nil, fmt.Errorf("unavailable") })})
	if _, err := c.Search("planned", 1); err == nil {
		t.Fatal("provider error lost")
	}
}

// FindTitle returns the snapshot saved by the metadata service.
func (m *metadataCache) FindTitle(context.Context, string, uint64) (*entity.LetterboxdTitle, error) {
	return m.title, nil
}

// SaveTitle captures successful provider snapshots.
func (m *metadataCache) SaveTitle(_ context.Context, title *entity.LetterboxdTitle) error {
	m.title, m.saves = title, m.saves+1
	return nil
}

// TestPlannedMovieCache reproduces the undated Maze Runner selection and repeated requests.
func TestPlannedMovieCache(t *testing.T) {
	calls := 0
	c, err := New("secret", http.Client{Transport: transport(func(*http.Request) (*http.Response, error) {
		calls++
		body := `{"adult":false,"backdrop_path":null,"belongs_to_collection":null,"budget":0,"genres":[{"id":878,"name":"Science Fiction"},{"id":28,"name":"Action"}],"homepage":"","id":1284679,"imdb_id":"tt32276864","origin_country":["US"],"original_language":"en","original_title":"Untitled Maze Runner Reboot","overview":"A somewhat continuation of the franchise. Plot TBA.","popularity":0.7065,"poster_path":null,"production_companies":[{"id":127928,"logo_path":"/h0rjX5vjW5r8yEnUBStFarjcLT4.png","name":"20th Century Studios","origin_country":"US"},{"id":3672,"logo_path":"/xpRwVe9NL5791oyUidzeRp8ezGj.png","name":"The Gotham Group","origin_country":"US"},{"id":12292,"logo_path":"/5J93SHsuvIV32AE9oSbMi02pqw9.png","name":"Temple Hill Entertainment","origin_country":"US"}],"production_countries":[{"iso_3166_1":"US","name":"United States of America"}],"release_date":"","revenue":0,"runtime":0,"softcore":false,"spoken_languages":[{"english_name":"English","iso_639_1":"en","name":"English"}],"status":"Planned","tagline":"","title":"Untitled Maze Runner Reboot","video":false,"vote_average":0.0,"vote_count":0}`
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
	})})
	if err != nil {
		t.Fatal(err)
	}
	cache := &metadataCache{}
	service := app.New(cache, c, nil)
	for range 3 {
		title, stale, err := service.Title(context.Background(), "movie", 1284679)
		if err != nil || stale {
			t.Fatalf("selection: %v, stale %v", err, stale)
		}
		if title.ReleaseYear != nil || title.ReleaseDate != nil || title.RuntimeMinutes != nil || title.Status == nil || *title.Status != "Planned" || title.PosterPath != "" {
			t.Fatalf("planned metadata: %#v", title)
		}
	}
	if calls != 1 || cache.saves != 1 {
		t.Fatalf("provider calls=%d saves=%d", calls, cache.saves)
	}
}

// TestAdditionalMetadata checks every newly wired field for both media namespaces.
func TestAdditionalMetadata(t *testing.T) {
	for _, kind := range []string{"movie", "tv"} {
		t.Run(kind, func(t *testing.T) {
			c, _ := New("secret", http.Client{Transport: transport(func(*http.Request) (*http.Response, error) {
				body := `{"id":1,"original_title":"Movie","original_name":"Series","release_date":"2030","first_air_date":"2030-01-02","runtime":120,"episode_run_time":[45,50],"original_language":"en","origin_country":["US","BR"],"status":"In Production","number_of_episodes":0,"external_ids":{"imdb_id":"tt123","wikidata_id":"Q1","facebook_id":"fb","instagram_id":"ig","twitter_id":"tw","freebase_mid":"/m/a","freebase_id":"/en/a","tvdb_id":123,"tvrage_id":456}}`
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
			})})
			title, err := c.Details(kind, 1)
			if err != nil {
				t.Fatal(err)
			}
			if title.ReleaseYear == nil || *title.ReleaseYear != 2030 || title.Status == nil || *title.Status != "In Production" || title.OriginalLanguage == nil || *title.OriginalLanguage != "en" || len(title.OriginCountries) != 2 {
				t.Fatalf("scalar metadata: %#v", title)
			}
			for key, value := range map[string]string{"imdb_id": "tt123", "wikidata_id": "Q1", "facebook_id": "fb", "instagram_id": "ig", "twitter_id": "tw"} {
				if title.ExternalIDs[key] != value {
					t.Fatalf("external ID %s: %#v", key, title.ExternalIDs)
				}
			}
			if kind == "movie" {
				if title.RuntimeMinutes == nil || *title.RuntimeMinutes != 120 || title.EpisodeCount != nil || title.EpisodeRuntimeMinutes != nil || title.ReleaseDate != nil {
					t.Fatalf("movie metadata: %#v", title)
				}
			} else if title.RuntimeMinutes != nil || title.EpisodeCount == nil || *title.EpisodeCount != 0 || len(title.EpisodeRuntimeMinutes) != 2 || title.EpisodeRuntimeMinutes[1] != 50 || title.ReleaseDate == nil || title.ExternalIDs["tvdb_id"] != "123" || title.ExternalIDs["tvrage_id"] != "456" || title.ExternalIDs["freebase_mid"] != "/m/a" || title.ExternalIDs["freebase_id"] != "/en/a" {
				t.Fatalf("TV metadata: %#v", title)
			}
		})
	}
}

// TestOptionalDates distinguishes unknown dates from malformed provider data.
func TestOptionalDates(t *testing.T) {
	for _, kind := range []string{"movie", "tv"} {
		for _, date := range []string{"", "null", `""`, `"garbage"`, `"0000"`, `"2026-02-30"`} {
			c, _ := New("secret", http.Client{Transport: transport(func(*http.Request) (*http.Response, error) {
				body := `{"id":1,"original_title":"Movie","original_name":"TV","external_ids":{}`
				if date != "" {
					body += `,"release_date":` + date + `,"first_air_date":` + date
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body + "}"))}, nil
			})})
			title, err := c.Details(kind, 1)
			invalid := date == `"garbage"` || date == `"0000"` || date == `"2026-02-30"`
			if invalid {
				if err == nil {
					t.Fatalf("accepted invalid %s date %s", kind, date)
				}
			} else if err != nil || title.ReleaseYear != nil || title.ReleaseDate != nil || title.ExternalIDs == nil || len(title.ExternalIDs) != 0 || title.Status != nil || title.OriginalLanguage != nil || title.OriginCountries == nil {
				t.Fatalf("unknown %s date %s: %#v %v", kind, date, title, err)
			}
		}
	}
}
