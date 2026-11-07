package letterboxd

import (
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
	app "github.com/kiLeo13/SaladaFun/discord/padinho/internal/application/letterboxd"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/locale/ptbr"
)

// TestTitleCardMetadata verifies fields, links, ratings, artwork and the single edit entry point.
func TestTitleCardMetadata(t *testing.T) {
	accent := uint32(0x123456)
	for _, kind := range []string{"movie", "tv"} {
		for _, homepage := range []string{"https://example.test/title", "", "javascript:bad"} {
			title := &entity.LetterboxdTitle{ID: 7, TMDBID: 42, MediaType: kind, OriginalTitle: "Title", ReleaseYear: 2021, Creators: []string{"Ross Duffer", "Matt Duffer"}, Genres: []string{"Drama", "Mystery"}, Homepage: homepage, SeasonCount: 5, Accent: &accent, BackdropPath: "/back.jpg", PosterPath: "/poster.jpg"}
			card := titleCard(title, []app.Opinion{{Rating: 5}, {Rating: 4}}, kind+":42")
			checkPayload(t, card)
			embed := card.Embeds[0]
			wantURL := "https://www.themoviedb.org/" + kind + "/42"
			if homepage == "https://example.test/title" {
				wantURL = homepage
			}
			if embed.Author.URL != wantURL || embed.URL != "https://www.themoviedb.org/"+kind+"/42" {
				t.Fatal("author/title URLs")
			}
			if embed.Footer.Text != "Nota geral: 4,5/5 · avaliado 2 vezes" {
				t.Fatal(embed.Footer.Text)
			}
			fields := map[string]string{}
			for _, field := range embed.Fields {
				fields[field.Name] = field.Value
			}
			name := ptbr.LetterboxdDirectors
			if kind == "tv" {
				name = ptbr.LetterboxdCreators
				if fields[ptbr.LetterboxdSeasons] != "5" {
					t.Fatal("seasons")
				}
			} else if _, exists := fields[ptbr.LetterboxdSeasons]; exists {
				t.Fatal("movie has seasons")
			}
			if fields[name] != "Ross Duffer, Matt Duffer" || fields[ptbr.LetterboxdGenres] != "Drama, Mystery" || fields[ptbr.LetterboxdReleaseYear] != "2021" {
				t.Fatalf("metadata fields %#v", fields)
			}
			if embed.Color != int(accent) || embed.Image.URL != "https://image.tmdb.org/t/p/w780/back.jpg" || embed.Thumbnail != nil {
				t.Fatal("artwork/accent changed")
			}
			buttons := card.Components[0].(discordgo.ActionsRow).Components
			if len(card.Components) != 1 || len(buttons) != 3 || buttons[0].(discordgo.Button).CustomID != "lb.watch:7" {
				t.Fatal("button order")
			}
			for _, b := range buttons {
				if b.(discordgo.Button).Label == ptbr.LetterboxdEdit {
					t.Fatal("edit button still present")
				}
			}
			title.BackdropPath = ""
			title.Accent = nil
			card = titleCard(title, nil, kind+":42")
			if card.Embeds[0].Footer.Text != "Sem avaliações." || card.Embeds[0].Color != 0x5865F2 || card.Embeds[0].Thumbnail.URL != "https://image.tmdb.org/t/p/w185/poster.jpg" {
				t.Fatal("empty rating/poster fallback")
			}
		}
	}
}

// TestTitleCardFieldBudgets covers maximum metadata text and absent optional fields.
func TestTitleCardFieldBudgets(t *testing.T) {
	title := &entity.LetterboxdTitle{ID: 1, TMDBID: 1, MediaType: "tv", OriginalTitle: strings.Repeat("🎬", 500), Synopsis: strings.Repeat("🎬", 5000), ReleaseYear: 2020, Creators: []string{strings.Repeat("*", 4000)}, Genres: []string{strings.Repeat("🎬", 4000)}}
	checkPayload(t, titleCard(title, []app.Opinion{{Rating: 5}}, "tv:1"))
	title.Creators = nil
	title.Genres = nil
	card := titleCard(title, nil, "tv:1")
	checkPayload(t, card)
	if len(card.Embeds[0].Fields) != 2 {
		t.Fatal("empty optional fields rendered")
	}
}
