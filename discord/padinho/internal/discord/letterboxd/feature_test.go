package letterboxd

import (
	"context"
	"errors"
	"testing"

	"github.com/bwmarrin/discordgo"
	app "github.com/kiLeo13/SaladaFun/discord/padinho/internal/application/letterboxd"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/command"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/discord"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
)

type catalog struct {
	items    []entity.LetterboxdTitle
	writes   int
	searches int
	offline  bool
	owner    uint64
}

// FindTitle supplies the complete local cache.
func (c *catalog) FindTitle(context.Context, string, uint64) (*entity.LetterboxdTitle, error) {
	return nil, nil
}

// SaveTitle captures full-detail cache writes.
func (c *catalog) SaveTitle(_ context.Context, t *entity.LetterboxdTitle) error {
	c.writes++
	t.ID = 1
	return nil
}

// Wishlist supplies the caller's stored interests.
func (c *catalog) Wishlist(_ context.Context, user uint64) ([]entity.LetterboxdTitle, error) {
	c.owner = user
	return c.items, nil
}

// AddWishlist records only the supplied actor.
func (c *catalog) AddWishlist(_ context.Context, user, title uint64) error {
	c.owner = user
	c.writes++
	return nil
}

// RemoveWishlist captures owner-scoped removal.
func (c *catalog) RemoveWishlist(_ context.Context, user, title uint64) (bool, error) {
	c.owner = user
	c.writes++
	return true, nil
}

// CachedSearch supplies complete fallback matches.
func (c *catalog) CachedSearch(context.Context, string) ([]entity.LetterboxdTitle, error) {
	return c.items, nil
}

// Reviews supplies current opinions.
func (c *catalog) Reviews(context.Context, uint64) ([]app.Opinion, error) { return nil, nil }

// Details supplies one complete selected title.
func (c *catalog) Details(k string, id uint64) (*entity.LetterboxdTitle, error) {
	return &entity.LetterboxdTitle{MediaType: k, TMDBID: id, OriginalTitle: "Original", ReleaseYear: 2020}, nil
}

// Search tracks remote access independently of persistence.
func (c *catalog) Search(string, int) ([]entity.LetterboxdTitle, error) {
	c.searches++
	if c.offline {
		return nil, errors.New("offline")
	}
	return []entity.LetterboxdTitle{{MediaType: "movie", TMDBID: 1, OriginalTitle: "First"}, {MediaType: "tv", TMDBID: 2, OriginalTitle: "Second"}}, nil
}

type responder struct {
	initial int
	data    *discordgo.InteractionResponseData
}

// Respond captures exactly one initial acknowledgement.
func (r *responder) Respond(v *discordgo.InteractionResponse) error {
	r.initial++
	r.data = v.Data
	return nil
}

// Complete captures the deferred final payload.
func (r *responder) Complete(v *discordgo.InteractionResponseData) error { r.data = v; return nil }

// TestDiscoveryFlows verifies search selection, local removal, and owner isolation.
func TestDiscoveryFlows(t *testing.T) {
	c := &catalog{}
	h := Handler{&app.Discovery{Metadata: app.New(c, c, nil), Catalog: c, SearchProvider: c}}
	ctx := context.Background()
	routes := discord.NewRoutes()
	Register(routes, h.service)
	if err := routes.Freeze(); err != nil {
		t.Fatal(err)
	}
	r := &responder{}
	request := &command.CommandRequest{GuildID: "1", Actor: command.Actor{UserID: "11"}, Responder: r, Options: command.NewOptionValues(map[string]any{"title": "query"})}
	if err := h.run("show")(ctx, request); err != nil || c.writes != 0 || len(r.data.Components) != 2 {
		t.Fatalf("search %v", err)
	}
	request.Options = command.NewOptionValues(map[string]any{"title": "movie:1"})
	if err := h.run("add")(ctx, request); err != nil || c.owner != 11 || c.writes != 2 || len(r.data.Embeds) != 1 {
		t.Fatalf("add %v", err)
	}
	c.items = []entity.LetterboxdTitle{{ID: 1, MediaType: "movie", TMDBID: 1, OriginalTitle: "First"}}
	before := c.searches
	request.Options = command.NewOptionValues(map[string]any{"title": "movie:1"})
	if err := h.run("remove")(ctx, request); err != nil || c.searches != before {
		t.Fatal("local removal")
	}
	auto := &discord.AutocompleteRequest{CommandRequest: request, Focused: "title"}
	request.Options = command.NewOptionValues(map[string]any{"title": "Fi"})
	before = c.writes
	if err := h.autocomplete("add")(ctx, auto); err != nil || len(r.data.Choices) != 1 || c.writes != before {
		t.Fatal("autocomplete filter")
	}
	if err := h.run("list")(ctx, request); err != nil {
		t.Fatal(err)
	}
	c.offline = true
	if _, cached, err := h.service.Search(ctx, 11, "show", "First", 1); err != nil || !cached {
		t.Fatal("fallback")
	}
	request.GuildID = ""
	if err := h.run("show")(ctx, request); err == nil {
		t.Fatal("DM")
	}
	for _, s := range []string{"movie:0", "person:1", "movie:abc", "broken"} {
		if _, _, err := app.ParseIdentity(s); err == nil {
			t.Fatal("identity")
		}
	}
}

// TestComponentValidation checks forged routes and deferred payload construction.
func TestComponentValidation(t *testing.T) {
	c := &catalog{}
	h := Handler{&app.Discovery{Metadata: app.New(c, c, nil), Catalog: c, SearchProvider: c}}
	r := &discord.InteractionRequest{GuildID: "1", Actor: command.Actor{UserID: "11"}, Responder: &responder{}}
	for _, f := range []discord.InteractionHandler{h.selectTitle, h.changePage, h.add, h.reviews} {
		if err := f(context.Background(), r); err == nil {
			t.Fatal("forged route")
		}
	}
	if clip("abc", 2) != "a\u2026" {
		t.Fatal("clip")
	}
}
