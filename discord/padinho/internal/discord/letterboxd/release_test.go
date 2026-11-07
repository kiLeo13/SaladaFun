package letterboxd

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf16"

	"github.com/bwmarrin/discordgo"
	app "github.com/kiLeo13/SaladaFun/discord/padinho/internal/application/letterboxd"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/command"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/discord"
)

// TestReleasePayloads checks compiled declarations and JSON emitted by local pages.
func TestReleasePayloads(t *testing.T) {
	c := &catalog{}
	service := &app.Discovery{Catalog: c, Metadata: app.New(c, c, nil), SearchProvider: c}
	routes := discord.NewRoutes()
	Register(routes, service)
	if err := routes.Freeze(); err != nil {
		t.Fatal(err)
	}
	definitions, err := routes.Commands().Definitions()
	if err != nil {
		t.Fatal(err)
	}
	commands, err := discord.CompileDefinitions(definitions)
	if err != nil || len(commands) != 1 {
		t.Fatal("compiled commands", err)
	}
	if len(commands[0].Options) != 5 {
		t.Fatalf("root routes: %d", len(commands[0].Options))
	}
	h := Handler{service}
	out := &responder{}
	for _, action := range []string{"list", "remove"} {
		if err := h.page(context.Background(), out, 11, action, "", 1); err != nil {
			t.Fatal(err)
		}
		payload, err := json.Marshal(out.data)
		if err != nil {
			t.Fatal(err)
		}
		var decoded struct {
			Embeds []*discordgo.MessageEmbed `json:"embeds"`
		}
		if err := json.Unmarshal(payload, &decoded); err != nil {
			t.Fatal(err)
		}
		for _, embed := range decoded.Embeds {
			if embed.Footer != nil && embed.Footer.Text == "" {
				t.Fatal("invalid empty embed footer")
			}
		}
	}
}

type reviewCatalog struct {
	*catalog
	opinions []app.Opinion
}

// Reviews supplies maximum-length current reviews without provider calls.
func (c reviewCatalog) Reviews(context.Context, uint64) ([]app.Opinion, error) {
	return c.opinions, nil
}

// TestLongReviewPayload preserves complete reviews within embed text limits.
func TestLongReviewPayload(t *testing.T) {
	text := strings.Repeat("*", 500)
	c := reviewCatalog{catalog: &catalog{}, opinions: []app.Opinion{{UserID: 1, Rating: 5, Review: &text}, {UserID: 2, Rating: 3, Review: &text}, {UserID: 3, Rating: 4, Review: &text}}}
	h := Handler{&app.Discovery{Catalog: c}}
	out := &responder{}
	request := &discord.InteractionRequest{GuildID: "1", Actor: command.Actor{UserID: "1"}, Parameters: []string{"1", "1"}, Responder: out}
	if err := h.reviews(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if len(out.data.Embeds) != 1 {
		t.Fatal("reviews must use an embed")
	}
	description := out.data.Embeds[0].Description
	if strings.Count(description, plainOpinion(text)) != 3 || len(utf16.Encode([]rune(description))) > 4096 {
		t.Fatal("review truncation or overflow")
	}
}
