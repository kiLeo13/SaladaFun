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
			title := &entity.LetterboxdTitle{ID: 7, TMDBID: 42, MediaType: kind, OriginalTitle: "Title", ReleaseYear: new(2021), Creators: []string{"Ross Duffer", "Matt Duffer"}, Genres: []string{"Drama", "Mystery"}, Homepage: homepage, SeasonCount: 5, Accent: &accent, BackdropPath: "/back.jpg", PosterPath: "/poster.jpg"}
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
			if embed.Footer.Text != "Nota Geral: 4,5/5 · avaliado 2 vezes" {
				t.Fatal(embed.Footer.Text)
			}
			fields := map[string]string{}
			for _, field := range embed.Fields {
				fields[field.Name] = field.Value
			}
			name := ptbr.LetterboxdDirectors
			release := "2021"
			if kind == "tv" {
				name = ptbr.LetterboxdCreators
				release = "2021 - 5 temporadas"
			}
			if fields[name] != "Ross Duffer, Matt Duffer" || fields[ptbr.LetterboxdGenres] != "Drama, Mystery" || fields[ptbr.LetterboxdReleaseYear] != release {
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
	title := &entity.LetterboxdTitle{ID: 1, TMDBID: 1, MediaType: "tv", OriginalTitle: strings.Repeat("🎬", 500), Synopsis: strings.Repeat("🎬", 5000), ReleaseYear: new(2020), Creators: []string{strings.Repeat("*", 4000)}, Genres: []string{strings.Repeat("🎬", 4000)}}
	checkPayload(t, titleCard(title, []app.Opinion{{Rating: 5}}, "tv:1"))
	title.Creators = nil
	title.Genres = nil
	card := titleCard(title, nil, "tv:1")
	checkPayload(t, card)
	if len(card.Embeds[0].Fields) != 1 {
		t.Fatal("empty optional fields rendered")
	}
}

// TestRatingFooter checks whole averages, decimal rounding, and singular Portuguese text.
func TestRatingFooter(t *testing.T) {
	for _, tc := range []struct {
		ratings []int
		want    string
	}{
		{nil, "Sem avaliações."},
		{[]int{5}, "Nota Geral: 5/5 · avaliado 1 vez"},
		{[]int{1}, "Nota Geral: 1/5 · avaliado 1 vez"},
		{[]int{5, 5}, "Nota Geral: 5/5 · avaliado 2 vezes"},
		{[]int{3, 5}, "Nota Geral: 4/5 · avaliado 2 vezes"},
		{[]int{4, 5}, "Nota Geral: 4,5/5 · avaliado 2 vezes"},
		{[]int{4, 5, 5}, "Nota Geral: 4,7/5 · avaliado 3 vezes"},
		{[]int{5, 5, 5, 4, 4, 4, 4}, "Nota Geral: 4,4/5 · avaliado 7 vezes"},
	} {
		opinions := make([]app.Opinion, len(tc.ratings))
		for i, rating := range tc.ratings {
			opinions[i].Rating = rating
		}
		card := titleCard(&entity.LetterboxdTitle{ID: 1, TMDBID: 1, MediaType: "movie", OriginalTitle: "Title", ReleaseYear: new(2020)}, opinions, "movie:1")
		checkPayload(t, card)
		if got := card.Embeds[0].Footer.Text; got != tc.want {
			t.Fatalf("ratings %v: %q, want %q", tc.ratings, got, tc.want)
		}
	}
}

// TestSynopsisEllipsis enforces the visible description budget at ASCII and Unicode boundaries.
func TestSynopsisEllipsis(t *testing.T) {
	for _, tc := range []struct{ synopsis, want string }{
		{"", ""},
		{strings.Repeat("a", 2000), strings.Repeat("a", 2000)},
		{strings.Repeat("a", 2001), strings.Repeat("a", 1997) + "..."},
		{strings.Repeat("🎬", 1000), strings.Repeat("🎬", 1000)},
		{strings.Repeat("🎬", 1001), strings.Repeat("🎬", 998) + "..."},
	} {
		card := titleCard(&entity.LetterboxdTitle{ID: 1, TMDBID: 1, OriginalTitle: "Title", MediaType: "movie", ReleaseYear: new(2020), Synopsis: tc.synopsis}, nil, "movie:1")
		checkPayload(t, card)
		if got := card.Embeds[0].Description; got != tc.want {
			t.Fatalf("synopsis clipping: got %q, want %q", got, tc.want)
		}
	}
}

// TestReleaseFieldSeasons verifies media-specific formatting and missing season counts.
func TestReleaseFieldSeasons(t *testing.T) {
	for _, tc := range []struct {
		kind    string
		seasons int
		want    string
	}{
		{"movie", 0, "2018"}, {"movie", 8, "2018"},
		{"tv", 0, "2018"}, {"tv", 1, "2018 - 1 temporada"},
		{"tv", 8, "2018 - 8 temporadas"},
	} {
		title := &entity.LetterboxdTitle{ID: 1, TMDBID: 1, MediaType: tc.kind, OriginalTitle: "Title", ReleaseYear: new(2018), SeasonCount: tc.seasons}
		card := titleCard(title, nil, tc.kind+":1")
		checkPayload(t, card)
		if got := card.Embeds[0].Fields[0]; got.Name != ptbr.LetterboxdReleaseYear || got.Value != tc.want {
			t.Fatalf("%s/%d release = %#v, want %q", tc.kind, tc.seasons, got, tc.want)
		}
		for _, field := range card.Embeds[0].Fields {
			if field.Name == "Temporadas" {
				t.Fatal("separate season field still rendered")
			}
		}
	}
}
