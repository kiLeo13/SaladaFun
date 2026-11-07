package letterboxd

import (
	"context"
	"errors"
	"testing"

	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
)

type interests struct {
	items                       []entity.LetterboxdTitle
	cached                      []entity.LetterboxdTitle
	listErr, cacheErr, writeErr error
	writes                      int
}

// Wishlist returns isolated caller interests.
func (i *interests) Wishlist(context.Context, uint64) ([]entity.LetterboxdTitle, error) {
	return i.items, i.listErr
}

// CachedSearch returns only complete fallback fixtures.
func (i *interests) CachedSearch(context.Context, string) ([]entity.LetterboxdTitle, error) {
	return i.cached, i.cacheErr
}

// AddWishlist captures explicit selected-title mutations.
func (i *interests) AddWishlist(context.Context, uint64, uint64) error { i.writes++; return i.writeErr }

// RemoveWishlist captures owner-scoped removals without provider access.
func (i *interests) RemoveWishlist(context.Context, uint64, uint64) (bool, error) {
	i.writes++
	return i.writeErr == nil, i.writeErr
}

// Reviews satisfies the public read-only opinion boundary.
func (i *interests) Reviews(context.Context, uint64) ([]Opinion, error) { return nil, nil }

type searches struct {
	items []entity.LetterboxdTitle
	err   error
	calls int
}

// Search tracks whether local-only actions accidentally contact the provider.
func (s *searches) Search(string, int) ([]entity.LetterboxdTitle, error) {
	s.calls++
	return s.items, s.err
}

// TestDiscoverySearch covers namespaces, local-only reads, fallback and error paths.
func TestDiscoverySearch(t *testing.T) {
	ctx := context.Background()
	movie := entity.LetterboxdTitle{ID: 1, TMDBID: 1, MediaType: "movie", OriginalTitle: "Movie"}
	tv := entity.LetterboxdTitle{ID: 2, TMDBID: 1, MediaType: "tv", OriginalTitle: "Series"}
	store := &interests{items: []entity.LetterboxdTitle{movie}}
	remote := &searches{items: []entity.LetterboxdTitle{movie, tv}}
	d := Discovery{Catalog: store, SearchProvider: remote}
	items, _, err := d.Search(ctx, 11, "add", "query", 1)
	if err != nil || len(items) != 1 || items[0].MediaType != "tv" || store.writes != 0 {
		t.Fatal("duplicate filter")
	}
	before := remote.calls
	for _, action := range []string{"remove", "list"} {
		items, _, err := d.Search(ctx, 11, action, "MOV", 1)
		if err != nil || len(items) != 1 || remote.calls != before {
			t.Fatal("local query")
		}
	}
	if items, _, err := d.Search(ctx, 11, "remove", "absent", 1); err != nil || len(items) != 0 {
		t.Fatal("local filter")
	}
	for _, page := range []int{0, 501} {
		if _, _, err := d.Search(ctx, 11, "show", "", page); err != ErrInvalid {
			t.Fatal("page")
		}
	}
	remote.err = errors.New("offline")
	store.cached = []entity.LetterboxdTitle{tv}
	items, cached, err := d.Search(ctx, 11, "show", "Series", 1)
	if err != nil || !cached || len(items) != 1 {
		t.Fatal("fallback")
	}
	store.cacheErr = errors.New("cache")
	if _, _, err := d.Search(ctx, 11, "show", "query", 1); err != store.cacheErr {
		t.Fatal("cache error")
	}
	store.cacheErr = nil
	store.listErr = errors.New("wishlist")
	if _, _, err := d.Search(ctx, 11, "remove", "query", 1); err != store.listErr {
		t.Fatal("local error")
	}
	remote.err = nil
	if _, _, err := d.Search(ctx, 11, "add", "query", 1); err != store.listErr {
		t.Fatal("filter error")
	}
}

// TestDiscoverySelection verifies explicit mutations and idempotent absent removals.
func TestDiscoverySelection(t *testing.T) {
	ctx := context.Background()
	store := &interests{}
	p := &provider{}
	d := Discovery{Catalog: store, Metadata: New(&memory{}, p, nil)}
	for _, v := range []struct {
		user          uint64
		action, value string
	}{{0, "show", "movie:1"}, {1, "bad", "movie:1"}, {1, "show", "bad"}} {
		if _, _, err := d.Select(ctx, v.user, v.action, v.value); err == nil {
			t.Fatal("selection validation")
		}
	}
	if _, _, err := d.Select(ctx, 1, "remove", "movie:1"); err != ErrWishlistAbsent || p.calls != 0 {
		t.Fatal("absent removal")
	}
	store.listErr = errors.New("list")
	if _, _, err := d.Select(ctx, 1, "remove", "movie:1"); err != store.listErr {
		t.Fatal("list error")
	}
	store.listErr = nil
	store.items = []entity.LetterboxdTitle{{ID: 1, TMDBID: 1, MediaType: "movie"}}
	if _, _, err := d.Select(ctx, 1, "remove", "movie:1"); err != nil || store.writes != 1 {
		t.Fatal("remove")
	}
	store.writeErr = errors.New("write")
	if _, _, err := d.Select(ctx, 1, "remove", "movie:1"); err != store.writeErr {
		t.Fatal("remove failure")
	}
	if _, _, err := d.Select(ctx, 1, "add", "tv:1"); err != store.writeErr {
		t.Fatal("add failure")
	}
	store.writeErr = nil
	if _, _, err := d.Select(ctx, 1, "show", "tv:1"); err != nil {
		t.Fatal("show")
	}
	if _, _, err := ParseIdentity("movie:0"); err != ErrInvalid {
		t.Fatal("identity zero")
	}
	if _, _, err := ParseIdentity("movie:x"); err != ErrInvalid {
		t.Fatal("identity numeric")
	}
}
