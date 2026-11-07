package letterboxd

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
)

// ErrWishlistAbsent identifies an idempotent removal of an absent interest.
var ErrWishlistAbsent = errors.New("wishlist title absent")

// Searcher returns provider suggestions without persisting partial metadata.
type Searcher interface {
	Search(string, int) ([]entity.LetterboxdTitle, error)
}

// Catalog stores local interests and supplies cached fallback suggestions.
type Catalog interface {
	Wishlist(context.Context, uint64) ([]entity.LetterboxdTitle, error)
	AddWishlist(context.Context, uint64, uint64) error
	RemoveWishlist(context.Context, uint64, uint64) (bool, error)
	CachedSearch(context.Context, string) ([]entity.LetterboxdTitle, error)
	Reviews(context.Context, uint64) ([]Opinion, error)
}

// Opinion is the current public rating and optional review for one member.
type Opinion struct {
	UserID uint64
	Rating int
	Review *string
}

// Discovery coordinates remote search with personal wishlist ownership.
type Discovery struct {
	Metadata       *Service
	Activity       *Activity
	Catalog        Catalog
	SearchProvider Searcher
}

// Search filters wishlist additions and uses complete local metadata on outages.
func (d *Discovery) Search(ctx context.Context, user uint64, action, query string, page int) ([]entity.LetterboxdTitle, bool, error) {
	query = strings.TrimSpace(query)
	if page < 1 || page > 500 {
		return nil, false, ErrInvalid
	}
	var items []entity.LetterboxdTitle
	var err error
	cached := false
	if action == "remove" || action == "list" {
		items, err = d.Catalog.Wishlist(ctx, user)
	} else {
		items, err = d.SearchProvider.Search(query, page)
		if err != nil {
			items, err = d.Catalog.CachedSearch(ctx, query)
			cached = true
		}
	}
	if err != nil {
		return nil, cached, err
	}
	existing := map[string]bool{}
	if action == "add" {
		list, e := d.Catalog.Wishlist(ctx, user)
		if e != nil {
			return nil, cached, e
		}
		for _, t := range list {
			existing[Identity(t.MediaType, t.TMDBID)] = true
		}
	}
	result := make([]entity.LetterboxdTitle, 0, len(items))
	for _, t := range items {
		if existing[Identity(t.MediaType, t.TMDBID)] {
			continue
		}
		if (action == "remove" || action == "list") && !strings.Contains(strings.ToLower(DisplayTitle(&t)), strings.ToLower(query)) {
			continue
		}
		result = append(result, t)
	}
	return result, cached, nil
}

// Identity creates a stable typed TMDB selection value.
func Identity(kind string, id uint64) string { return kind + ":" + strconv.FormatUint(id, 10) }

// ParseIdentity validates both the namespace and numeric selection.
func ParseIdentity(value string) (string, uint64, error) {
	parts := strings.Split(value, ":")
	if len(parts) != 2 {
		return "", 0, ErrInvalid
	}
	id, err := strconv.ParseUint(parts[1], 10, 64)
	if err != nil || !ValidIdentity(parts[0], id) {
		return "", 0, ErrInvalid
	}
	return parts[0], id, nil
}

// Select resolves metadata for show/add; removal is entirely local.
func (d *Discovery) Select(ctx context.Context, user uint64, action, value string) (*entity.LetterboxdTitle, bool, error) {
	if user == 0 || (action != "show" && action != "add" && action != "remove") {
		return nil, false, ErrInvalid
	}
	kind, id, err := ParseIdentity(value)
	if err != nil {
		return nil, false, err
	}
	if action == "remove" {
		items, e := d.Catalog.Wishlist(ctx, user)
		if e != nil {
			return nil, false, e
		}
		for _, t := range items {
			if t.MediaType == kind && t.TMDBID == id {
				_, e = d.Catalog.RemoveWishlist(ctx, user, t.ID)
				return &t, false, e
			}
		}
		return nil, false, ErrWishlistAbsent
	}
	title, stale, err := d.Metadata.Title(ctx, kind, id)
	if err == nil && action == "add" {
		err = d.Catalog.AddWishlist(ctx, user, title.ID)
	}
	return title, stale, err
}
