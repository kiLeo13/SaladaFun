// Package letterboxd owns catalog freshness and individual viewing rules.
package letterboxd

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
)

const MetadataTTL = 24 * time.Hour

var ErrInvalid = errors.New("invalid letterboxd input")

// Repository stores complete title snapshots.
type Repository interface {
	FindTitle(context.Context, string, uint64) (*entity.LetterboxdTitle, error)
	SaveTitle(context.Context, *entity.LetterboxdTitle) error
}

// Provider retrieves complete movie or TV metadata.
type Provider interface {
	Details(string, uint64) (*entity.LetterboxdTitle, error)
}

// Artwork extracts an optional accent from a validated CDN image path.
type Artwork interface {
	Accent(context.Context, string) (uint32, error)
}

// Service resolves full metadata on demand, retaining stale data on failures.
type Service struct {
	repository Repository
	provider   Provider
	artwork    Artwork
	now        func() time.Time
}

// New constructs the metadata service with an optional artwork extractor.
func New(repository Repository, provider Provider, artwork Artwork) *Service {
	return &Service{repository, provider, artwork, time.Now}
}

// Title reuses fresh metadata and refreshes only requested expired records.
// The boolean reports that stale metadata was returned after a provider failure.
func (s *Service) Title(ctx context.Context, kind string, id uint64) (*entity.LetterboxdTitle, bool, error) {
	if !ValidIdentity(kind, id) {
		return nil, false, ErrInvalid
	}
	cached, err := s.repository.FindTitle(ctx, kind, id)
	if err != nil {
		return nil, false, err
	}
	now := s.now()
	if cached != nil && cached.ContentRatings != nil && now.UnixMilli()-cached.MetadataFetchedAt < MetadataTTL.Milliseconds() {
		return cached, false, nil
	}
	title, err := s.provider.Details(kind, id)
	if err == nil && (title == nil || title.MediaType != kind || title.TMDBID != id || ValidateTitle(title) != nil) {
		err = ErrInvalid
	}
	if err != nil {
		if cached != nil {
			return cached, true, nil
		}
		return nil, false, err
	}
	source := title.BackdropPath
	if source == "" {
		source = title.PosterPath
	}
	if cached != nil && cached.AccentSource == source {
		title.Accent = cached.Accent
		title.AccentSource = cached.AccentSource
	}
	if title.Accent == nil && source != "" && s.artwork != nil {
		if accent, e := s.artwork.Accent(ctx, source); e == nil {
			title.Accent = &accent
			title.AccentSource = source
		}
	}
	title.MetadataFetchedAt = now.UnixMilli()
	if err = s.repository.SaveTitle(ctx, title); err != nil {
		return nil, false, err
	}
	return title, false, nil
}

// ValidIdentity restricts identities to supported TMDB media namespaces.
func ValidIdentity(kind string, id uint64) bool { return id > 0 && (kind == "movie" || kind == "tv") }

// ValidateTitle prevents incomplete or oversized snapshots from entering cache.
func ValidateTitle(t *entity.LetterboxdTitle) error {
	if t.SeasonCount < 0 || utf8.RuneCountInString(t.Homepage) > 2048 {
		return ErrInvalid
	}
	if !ValidIdentity(t.MediaType, t.TMDBID) || strings.TrimSpace(t.OriginalTitle) == "" || utf8.RuneCountInString(t.OriginalTitle) > 500 || utf8.RuneCountInString(t.LocalizedTitle) > 500 || t.ReleaseYear < 1 || t.ReleaseYear > 9999 {
		return ErrInvalid
	}
	return nil
}

// MetadataNames trims and deduplicates names while preserving punctuation and order.
// A non-nil empty slice serializes to a JSON array rather than JSON null.
func MetadataNames(names []string) []string {
	result := make([]string, 0, len(names))
	seen := map[string]bool{}
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name != "" && !seen[name] {
			result = append(result, name)
			seen[name] = true
		}
	}
	return result
}

// HomepageURL accepts only absolute HTTP(S) links suitable for Discord authors.
func HomepageURL(raw string) string {
	raw = strings.TrimSpace(raw)
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" || u.User != nil || (u.Scheme != "https" && u.Scheme != "http") || utf8.RuneCountInString(raw) > 2048 {
		return ""
	}
	return raw
}

// DisplayTitle includes the localized name only when different and available.
func DisplayTitle(t *entity.LetterboxdTitle) string {
	original, local := strings.TrimSpace(t.OriginalTitle), strings.TrimSpace(t.LocalizedTitle)
	if local != "" && local != original {
		return original + " (" + local + ")"
	}
	return original
}

// ReviewText normalizes optional text; nil represents withdrawal.
func ReviewText(text string) (*string, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, nil
	}
	n := utf8.RuneCountInString(text)
	if n < 2 || n > 500 {
		return nil, ErrInvalid
	}
	return &text, nil
}
