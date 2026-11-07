package letterboxd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/bwmarrin/discordgo"
	app "github.com/kiLeo13/SaladaFun/discord/padinho/internal/application/letterboxd"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/command"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/discord"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
)

type boundaryHistoryStore struct{ *activityStore }

// History supplies a full page of maximum-length escaped titles and opinions.
func (s boundaryHistoryStore) History(context.Context, uint64, string, string, int) ([]app.HistoryEntry, error) {
	text := strings.Repeat("*", 500)
	entries := make([]app.HistoryEntry, app.HistoryPageSize+1)
	for i := range entries {
		entries[i] = app.HistoryEntry{OriginalTitle: strings.Repeat("🎬", 500), MediaType: "movie", Review: &text, Rating: 5}
	}
	return entries, nil
}

// checkPayload validates the serialized shapes and limits used by Letterboxd.
func checkPayload(t *testing.T, data *discordgo.InteractionResponseData) {
	t.Helper()
	raw, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatal(err)
	}
	length := func(s string) int { return len(utf16.Encode([]rune(s))) }
	bound := func(s string, min, max int) {
		t.Helper()
		if n := length(s); n < min || n > max {
			t.Fatalf("text length %d outside %d..%d", n, min, max)
		}
	}
	bound(data.Content, 0, 2000)
	if len(data.Embeds) > 10 {
		t.Fatal("too many embeds")
	}
	total := 0
	for _, e := range data.Embeds {
		bound(e.Title, 0, 256)
		bound(e.Description, 0, 4096)
		total += length(e.Title) + length(e.Description)
		if e.Footer != nil {
			bound(e.Footer.Text, 1, 2048)
			total += length(e.Footer.Text)
		}
	}
	if total > 6000 {
		t.Fatal("aggregate embed limit")
	}
	if len(data.Choices) > 25 {
		t.Fatal("autocomplete choice limit")
	}
	for _, c := range data.Choices {
		bound(c.Name, 1, 100)
		bound(c.Value.(string), 1, 100)
	}
	modal := data.CustomID != ""
	if modal {
		bound(data.CustomID, 1, 100)
		bound(data.Title, 1, 45)
	}
	ids := map[string]bool{}
	var visit func(map[string]any)
	visit = func(c map[string]any) {
		if id, ok := c["custom_id"].(string); ok {
			bound(id, 1, 100)
			if ids[id] {
				t.Fatalf("duplicate custom_id %q", id)
			}
			ids[id] = true
		}
		switch int(c["type"].(float64)) {
		case 1:
			children := c["components"].([]any)
			if modal || len(children) < 1 || len(children) > 5 {
				t.Fatal("invalid action row")
			}
			for _, child := range children {
				kind := int(child.(map[string]any)["type"].(float64))
				if kind != 2 && (kind != 3 || len(children) != 1) {
					t.Fatal("invalid row child")
				}
				visit(child.(map[string]any))
			}
		case 2:
			bound(c["label"].(string), 1, 80)
			if style := c["style"].(float64); style < 1 || style > 4 {
				t.Fatal("button style")
			}
		case 3:
			options := c["options"].([]any)
			if len(options) < 1 || len(options) > 25 {
				t.Fatal("select option count")
			}
			values := map[string]bool{}
			defaults := 0
			for _, item := range options {
				o := item.(map[string]any)
				value := o["value"].(string)
				bound(o["label"].(string), 1, 100)
				bound(value, 1, 100)
				if values[value] {
					t.Fatal("duplicate select value")
				}
				values[value] = true
				if o["default"] == true {
					defaults++
				}
			}
			if defaults > 1 {
				t.Fatal("single select defaults")
			}
			if p, ok := c["placeholder"].(string); ok {
				bound(p, 0, 150)
			}
		case 18:
			if !modal {
				t.Fatal("label outside modal")
			}
			bound(c["label"].(string), 1, 45)
			if d, ok := c["description"].(string); ok {
				bound(d, 0, 100)
			}
			visit(c["component"].(map[string]any))
		case 4:
			if !modal {
				t.Fatal("text input outside modal")
			}
			if v, ok := c["value"].(string); ok {
				bound(v, 0, 4000)
			}
			if c["max_length"].(float64) > 4000 {
				t.Fatal("text input maximum")
			}
		default:
			t.Fatal("unexpected component type")
		}
	}
	if components, ok := payload["components"].([]any); ok {
		if len(components) > 5 {
			t.Fatal("top-level component limit")
		}
		for _, c := range components {
			visit(c.(map[string]any))
		}
	}
}

// TestLetterboxdPayloadConstraints exercises all renderers with Unicode and boundary pages.
func TestLetterboxdPayloadConstraints(t *testing.T) {
	ctx := context.Background()
	title := entity.LetterboxdTitle{ID: ^uint64(0), TMDBID: ^uint64(0), MediaType: "movie", OriginalTitle: strings.Repeat("🎬", 500), Synopsis: strings.Repeat("🎬", 5000)}
	card := titleCard(&title, nil, app.Identity(title.MediaType, title.TMDBID))
	checkPayload(t, card)
	buttons := card.Components[1].(discordgo.ActionsRow).Components
	if buttons[0].(discordgo.Button).CustomID != fmt.Sprintf("lb.watch:%d", title.ID) || buttons[1].(discordgo.Button).CustomID != fmt.Sprintf("lb.edit:%d", title.ID) {
		t.Fatal("watch/edit targets")
	}
	items := []entity.LetterboxdTitle{title, title}
	for i := uint64(1); i <= 30; i++ {
		item := title
		item.TMDBID = i
		items = append(items, item)
	}
	for _, action := range []string{"show", "add", "remove", "list"} {
		for _, page := range []int{1, 2, 500} {
			checkPayload(t, resultPage(action, strings.Repeat("🎬", 200), page, items, true, true))
			checkPayload(t, resultPage(action, "", page, nil, false, false))
		}
	}
	c := &catalog{items: items}
	store := &activityStore{}
	h := Handler{&app.Discovery{Catalog: c, Metadata: app.New(c, c, nil), SearchProvider: c, Activity: &app.Activity{Store: store}}}
	routes := discord.NewRoutes()
	Register(routes, h.service)
	if err := routes.Freeze(); err != nil {
		t.Fatal(err)
	}
	out := &responder{}
	request := &command.CommandRequest{GuildID: "1", Actor: command.Actor{UserID: "11"}, Responder: out, Options: command.NewOptionValues(map[string]any{"title": ""})}
	if err := h.autocomplete("remove")(ctx, &discord.AutocompleteRequest{CommandRequest: request, Focused: "title"}); err != nil {
		t.Fatal(err)
	}
	checkPayload(t, out.data)
	if len(out.data.Choices) != 25 {
		t.Fatal("autocomplete should cap at 25 choices")
	}
	if err := h.about(ctx, request); err != nil {
		t.Fatal(err)
	}
	checkPayload(t, out.data)
	for _, user := range []command.Snowflake{"11", "12"} {
		if err := h.openWatch(ctx, &discord.InteractionRequest{GuildID: "1", Actor: command.Actor{UserID: user}, Responder: out, Parameters: []string{"1"}}); err != nil {
			t.Fatal(err)
		}
		checkPayload(t, out.data)
	}
	for _, empty := range []bool{false, true} {
		store.empty = empty
		for _, page := range []int{1, 2, 100000} {
			if err := h.renderHistory(ctx, out, ^uint64(0), "rating", "desc", page); err != nil {
				t.Fatal(err)
			}
			checkPayload(t, out.data)
			if page == 100000 && !out.data.Components[0].(discordgo.ActionsRow).Components[1].(discordgo.Button).Disabled {
				t.Fatal("history upper bound")
			}
		}
	}
	for _, text := range []string{strings.Repeat("*", 500), strings.Repeat("🎬", 500)} {
		reviews := reviewCatalog{catalog: c, opinions: []app.Opinion{{UserID: ^uint64(0), Rating: 5, Review: &text}, {UserID: 2, Rating: 4, Review: &text}, {UserID: 3, Rating: 3, Review: &text}}}
		h.service.Catalog = reviews
		if err := h.reviews(ctx, &discord.InteractionRequest{GuildID: "1", Actor: request.Actor, Responder: out, Parameters: []string{"1", "1"}}); err != nil {
			t.Fatal(err)
		}
		checkPayload(t, out.data)
	}
	h.service.Activity.Store = boundaryHistoryStore{store}
	if err := h.renderHistory(ctx, out, 1, "date", "desc", 1); err != nil {
		t.Fatal(err)
	}
	checkPayload(t, out.data)
}

// TestClipUTF16 checks exact boundaries without corrupting multibyte characters.
func TestClipUTF16(t *testing.T) {
	for _, tc := range []struct {
		s    string
		n    int
		want string
	}{
		{"abc", 3, "abc"}, {"abc", 2, "a…"}, {"🎬🎬", 4, "🎬🎬"}, {"🎬🎬", 3, "🎬…"}, {"🎬", 1, "…"}, {"abc", 0, ""}, {"a🎬b", 3, "a…"},
	} {
		if got := clip(tc.s, tc.n); got != tc.want || !utf8.ValidString(got) {
			t.Fatalf("clip(%q,%d) = %q, want %q", tc.s, tc.n, got, tc.want)
		}
	}
}
