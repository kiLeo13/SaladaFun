package letterboxd

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
)

// TestMetadataHelpers preserves names and rejects unsuitable homepage URLs.
func TestMetadataHelpers(t *testing.T) {
	if got := MetadataNames([]string{" Ross Duffer ", "", "Matt Duffer", "Ross Duffer", "Name, Jr."}); !reflect.DeepEqual(got, []string{"Ross Duffer", "Matt Duffer", "Name, Jr."}) {
		t.Fatalf("names: %#v", got)
	}
	if got := MetadataNames(nil); got == nil || len(got) != 0 {
		t.Fatal("nil must become empty JSON array")
	}
	for _, raw := range []string{"", "/relative", "//example.test", "javascript:alert(1)", "https://user:pass@example.test", "https://", "https://example.test/\npath", "https://example.test/" + strings.Repeat("x", 2048)} {
		if HomepageURL(raw) != "" {
			t.Fatalf("invalid homepage accepted: %q", raw)
		}
	}
	for _, raw := range []string{"http://example.test/", "https://example.test/path?q=1"} {
		if HomepageURL(" "+raw+" ") != raw {
			t.Fatal("valid homepage")
		}
	}
	title := &entity.LetterboxdTitle{MediaType: "tv", TMDBID: 1, OriginalTitle: "TV", ReleaseYear: 2020, SeasonCount: -1}
	if ValidateTitle(title) == nil {
		t.Fatal("negative seasons")
	}
	title.SeasonCount = 0
	title.Homepage = strings.Repeat("x", 2049)
	if ValidateTitle(title) == nil {
		t.Fatal("oversized homepage")
	}
}

type memory struct {
	title   *entity.LetterboxdTitle
	saves   int
	saveErr error
	err     error
}

// FindTitle implements an isolated cache fake.
func (m *memory) FindTitle(context.Context, string, uint64) (*entity.LetterboxdTitle, error) {
	return m.title, m.err
}

// SaveTitle captures complete writes.
func (m *memory) SaveTitle(_ context.Context, t *entity.LetterboxdTitle) error {
	m.saves++
	m.title = t
	return m.saveErr
}

type provider struct {
	calls int
	err   error
}

// Details returns an independent valid response.
func (p *provider) Details(k string, id uint64) (*entity.LetterboxdTitle, error) {
	p.calls++
	return &entity.LetterboxdTitle{MediaType: k, TMDBID: id, OriginalTitle: "Original", ReleaseYear: 2020, BackdropPath: "/a.jpg", ContentRatings: []entity.ContentRating{}}, p.err
}

type artwork struct {
	path  string
	calls int
	err   error
}

// Accent captures extraction attempts.
func (a *artwork) Accent(_ context.Context, path string) (uint32, error) {
	a.path = path
	a.calls++
	return 0x112233, a.err
}

// TestCacheLifecycle covers freshness, expiration, accent reuse, and stale fallback.
func TestCacheLifecycle(t *testing.T) {
	ctx := context.Background()
	m := &memory{}
	p := &provider{}
	a := &artwork{}
	s := New(m, p, a)
	now := time.Unix(100000, 0)
	s.now = func() time.Time { return now }
	title, stale, err := s.Title(ctx, "movie", 1)
	if err != nil || stale || title.Accent == nil || m.saves != 1 || a.path != title.BackdropPath {
		t.Fatalf("first: %#v %v %v", title, stale, err)
	}
	s.Title(ctx, "movie", 1)
	if p.calls != 1 {
		t.Fatal("fresh fetch")
	}
	now = now.Add(MetadataTTL)
	s.Title(ctx, "movie", 1)
	if p.calls != 2 || a.calls != 1 {
		t.Fatal("TTL/accent reuse")
	}
	now = now.Add(MetadataTTL)
	p.err = errors.New("offline")
	before := m.title.MetadataFetchedAt
	_, stale, err = s.Title(ctx, "movie", 1)
	if err != nil || !stale || m.title.MetadataFetchedAt != before {
		t.Fatal("stale fallback")
	}
	m.title = nil
	if _, _, err = s.Title(ctx, "movie", 1); err == nil {
		t.Fatal("missing error")
	}
	if _, _, err = s.Title(ctx, "person", 1); err == nil {
		t.Fatal("identity")
	}
	m.err = errors.New("database")
	if _, _, err = s.Title(ctx, "movie", 1); err == nil {
		t.Fatal("database error")
	}
}

// TestValidation covers title formatting and Unicode review boundaries.
func TestValidation(t *testing.T) {
	title := &entity.LetterboxdTitle{OriginalTitle: "Original", LocalizedTitle: "Traduzido"}
	if DisplayTitle(title) != "Original (Traduzido)" {
		t.Fatal("display")
	}
	title.LocalizedTitle = "Original"
	if DisplayTitle(title) != "Original" {
		t.Fatal("duplicate")
	}
	for _, input := range []string{"a", strings.Repeat("?", 501)} {
		if _, err := ReviewText(input); err == nil {
			t.Fatal("review accepted")
		}
	}
	if r, err := ReviewText("  "); r != nil || err != nil {
		t.Fatal("blank")
	}
	if r, err := ReviewText(" ?? "); err != nil || *r != "??" {
		t.Fatal("unicode")
	}
	if ValidateTitle(title) == nil {
		t.Fatal("invalid title")
	}
}

type providerFunc func(string, uint64) (*entity.LetterboxdTitle, error)

// Details injects incomplete and alternate artwork fixtures.
func (f providerFunc) Details(k string, id uint64) (*entity.LetterboxdTitle, error) { return f(k, id) }

// TestCacheFailurePaths covers incomplete snapshots and persistence/image failures.
func TestCacheFailurePaths(t *testing.T) {
	ctx := context.Background()
	invalid := New(&memory{}, providerFunc(func(string, uint64) (*entity.LetterboxdTitle, error) { return nil, nil }), nil)
	if _, _, err := invalid.Title(ctx, "tv", 1); err == nil {
		t.Fatal("nil snapshot")
	}
	fixture := providerFunc(func(k string, id uint64) (*entity.LetterboxdTitle, error) {
		return &entity.LetterboxdTitle{MediaType: k, TMDBID: id, OriginalTitle: "Title", ReleaseYear: 2020, PosterPath: "/poster.jpg"}, nil
	})
	m := &memory{saveErr: errors.New("write")}
	s := New(m, fixture, nil)
	if _, _, err := s.Title(ctx, "tv", 1); err == nil {
		t.Fatal("write error")
	}
	m = &memory{}
	a := &artwork{err: errors.New("image")}
	s = New(m, fixture, a)
	title, _, err := s.Title(ctx, "tv", 1)
	if err != nil || title.Accent != nil || a.calls != 1 || a.path != title.PosterPath {
		t.Fatal("image failure")
	}
}
