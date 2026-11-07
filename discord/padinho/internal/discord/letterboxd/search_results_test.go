package letterboxd

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
	app "github.com/kiLeo13/SaladaFun/discord/padinho/internal/application/letterboxd"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/command"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/discord"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
)

type pagedSearch struct {
	*catalog
	offline       bool
	queries       []string
	fallbackPages []int
}

// Search returns a small filtered page with an independently large provider total.
func (s *pagedSearch) Search(query string, page int) (app.SearchResults, error) {
	s.queries = append(s.queries, query)
	if s.offline {
		return app.SearchResults{}, errors.New("offline")
	}
	return app.SearchResults{Items: []entity.LetterboxdTitle{{MediaType: "movie", TMDBID: 1, OriginalTitle: "Title"}}, TotalResults: 125, TotalPages: 7}, nil
}

// CachedSearch simulates database paging with one remaining item after page one.
func (s *pagedSearch) CachedSearch(_ context.Context, _ string, page int) (app.SearchResults, error) {
	s.fallbackPages = append(s.fallbackPages, page)
	return app.SearchResults{Items: []entity.LetterboxdTitle{{MediaType: "movie", TMDBID: 2, OriginalTitle: "Last cached title"}}, TotalResults: 21, TotalPages: 2}, nil
}

// TestResultTotalsAndPagination preserves query text and source totals across old/new buttons.
func TestResultTotalsAndPagination(t *testing.T) {
	ctx := context.Background()
	s := &pagedSearch{catalog: &catalog{}}
	h := Handler{&app.Discovery{Catalog: s, SearchProvider: s}}
	out := &responder{}
	query := "Consulta: title"
	if err := h.page(ctx, out, 11, "show", query, 1); err != nil {
		t.Fatal(err)
	}
	checkPayload(t, out.data)
	if got := out.data.Embeds[0]; got.Title != "Resultados (125)" || got.Footer.Text != "Consulta: Consulta: title" {
		t.Fatalf("remote heading %#v", got)
	}
	request := &discord.InteractionRequest{GuildID: "1", Actor: command.Actor{UserID: "11"}, Responder: out, Parameters: []string{"show", "2"}, Interaction: &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{Message: &discordgo.Message{Embeds: out.data.Embeds}}}}
	if err := h.changePage(ctx, request); err != nil {
		t.Fatal(err)
	}
	if s.queries[len(s.queries)-1] != query {
		t.Fatal("query prefix accumulated")
	}
	request.Interaction.Message.Embeds = []*discordgo.MessageEmbed{{Footer: &discordgo.MessageEmbedFooter{Text: query}}}
	if err := h.changePage(ctx, request); err != nil {
		t.Fatal(err)
	}
	if s.queries[len(s.queries)-1] != query {
		t.Fatal("legacy footer changed query")
	}
	if err := h.page(ctx, out, 11, "show", "title", 7); err != nil {
		t.Fatal(err)
	}
	next := out.data.Components[len(out.data.Components)-1].(discordgo.ActionsRow).Components[1].(discordgo.Button)
	if !next.Disabled {
		t.Fatal("next enabled beyond provider pages")
	}
	s.offline = true
	if err := h.page(ctx, out, 11, "show", "title", 2); err != nil {
		t.Fatal(err)
	}
	checkPayload(t, out.data)
	if !strings.HasPrefix(out.data.Embeds[0].Title, "⚠️ Resultados (21)") || len(out.data.Components) != 2 || out.data.Content != "" {
		t.Fatal("cached page was sliced twice or warning missing")
	}
	if len(s.fallbackPages) != 1 || s.fallbackPages[0] != 2 {
		t.Fatal("database page not forwarded")
	}
	if got := out.data.Components[0].(discordgo.ActionsRow).Components[0].(discordgo.SelectMenu); len(got.Options) != 1 {
		t.Fatal("cached page lost its only item")
	}
	for _, cached := range []bool{false, true} {
		data := resultPage("show", "query", 1, nil, cached, false, 0)
		checkPayload(t, data)
		if !strings.HasSuffix(data.Embeds[0].Title, "Resultados (0)") || data.Embeds[0].Footer.Text != "Consulta: query" {
			t.Fatal("empty results heading")
		}
	}
}
