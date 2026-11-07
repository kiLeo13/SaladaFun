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

type reviewsCatalog struct {
	reviewCatalog
	titleErr  error
	reviewErr error
}

// TestLegacyReviewPrefill flattens old multiline text in the edit modal without saving.
func TestLegacyReviewPrefill(t *testing.T) {
	text := " \r Old\n\n review \r\n "
	store := &activityStore{reviewText: &text}
	h := Handler{&app.Discovery{Activity: &app.Activity{Store: store}}}
	out := &responder{}
	r := &discord.InteractionRequest{GuildID: "1", Actor: command.Actor{UserID: "11"}, Responder: out, Parameters: []string{"1"}}
	if err := h.openWatch(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	checkPayload(t, out.data)
	field := out.data.Components[1].(discordgo.Label).Component.(discordgo.TextInput)
	if field.Value != "Old review" || store.writes != 0 || text != " \r Old\n\n review \r\n " {
		t.Fatal("legacy modal prefill", field.Value)
	}
}

// Reviews supplies database read failures independently of cached title lookup.
func (c reviewsCatalog) Reviews(context.Context, uint64) ([]app.Opinion, error) {
	return c.opinions, c.reviewErr
}

// TitleByID supplies local title lookup failures independently of opinion reads.
func (c reviewsCatalog) TitleByID(context.Context, uint64) (*entity.LetterboxdTitle, error) {
	return &entity.LetterboxdTitle{OriginalTitle: strings.Repeat("Title", 100)}, c.titleErr
}

// TestReviewsPresentation verifies headings, full totals, spacing and flattened quotes.
func TestReviewsPresentation(t *testing.T) {
	text := " \r\n Great\r\n\n  movie *wow* \r "
	blank := " \r\n\n "
	c := reviewsCatalog{reviewCatalog: reviewCatalog{catalog: &catalog{}, opinions: []app.Opinion{
		{UserID: 11, Rating: 5, Review: &text}, {UserID: 12, Rating: 4}, {UserID: 13, Rating: 3, Review: &blank}, {UserID: 14, Rating: 2},
	}}}
	h := Handler{&app.Discovery{Catalog: c}}
	out := &responder{}
	request := &discord.InteractionRequest{GuildID: "1", Actor: command.Actor{UserID: "11"}, Parameters: []string{"1", "1"}, Responder: out}
	if err := h.reviews(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	checkPayload(t, out.data)
	embed := out.data.Embeds[0]
	if !strings.HasPrefix(embed.Title, "Avaliações de Title") || embed.Footer.Text != "Total: 4 avaliações." {
		t.Fatalf("header/footer: %#v", embed)
	}
	want := "<@11>: **5/5**\n> Great movie \\*wow\\*\n\n<@12>: **4/5**\n> *Sem comentário*\n\n<@13>: **3/5**\n> *Sem comentário*"
	if embed.Description != want {
		t.Fatalf("description = %q, want %q", embed.Description, want)
	}
	if text != " \r\n Great\r\n\n  movie *wow* \r " {
		t.Fatal("render changed stored text")
	}
	request.Parameters[1] = "2"
	if err := h.reviews(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if got := out.data.Embeds[0]; got.Footer.Text != "Total: 4 avaliações." || got.Description != "<@14>: **2/5**\n> *Sem comentário*" {
		t.Fatalf("pagination %#v", got)
	}
	for _, count := range []int{0, 1} {
		c.opinions = c.opinions[:count]
		h.service.Catalog = c
		request.Parameters[1] = "1"
		if err := h.reviews(context.Background(), request); err != nil {
			t.Fatal(err)
		}
		checkPayload(t, out.data)
		want := "Total: 0 avaliações."
		if count == 1 {
			want = "Total: 1 avaliação."
		}
		if out.data.Embeds[0].Footer.Text != want {
			t.Fatal("total", count)
		}
	}
	c.titleErr = errors.New("database unavailable")
	h.service.Catalog = c
	if err := h.reviews(context.Background(), request); !errors.Is(err, c.titleErr) {
		t.Fatal("title error", err)
	}
}

// TestRenderedOpinionsNormalizeLegacyWhitespace covers shared reviews and personal history text.
func TestRenderedOpinionsNormalizeLegacyWhitespace(t *testing.T) {
	if got := plainOpinion(" \r\n A\rB\n\n C >x \r "); got != "A B C \\>x" {
		t.Fatal(got)
	}
}

// TestReviewsValidation rejects forged pages and propagates context/database failures.
func TestReviewsValidation(t *testing.T) {
	c := reviewsCatalog{reviewCatalog: reviewCatalog{catalog: &catalog{}}}
	h := Handler{&app.Discovery{Catalog: c}}
	r := &discord.InteractionRequest{GuildID: "1", Actor: command.Actor{UserID: "11"}, Responder: &responder{}}
	for _, parameters := range [][]string{nil, {"0", "1"}, {"invalid", "1"}, {"1", "invalid"}, {"1", "0"}, {"1", "100001"}, {"1", "2"}} {
		r.Parameters = parameters
		if err := h.reviews(context.Background(), r); err == nil {
			t.Fatal("invalid page accepted", parameters)
		}
	}
	r.Parameters = []string{"1", "1"}
	r.GuildID = ""
	if err := h.reviews(context.Background(), r); err == nil {
		t.Fatal("DM accepted")
	}
	r.GuildID = "1"
	c.reviewErr = errors.New("opinions read failed")
	h.service.Catalog = c
	if err := h.reviews(context.Background(), r); !errors.Is(err, c.reviewErr) {
		t.Fatal("review read error", err)
	}
}
