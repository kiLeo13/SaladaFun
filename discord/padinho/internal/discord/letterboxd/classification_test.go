package letterboxd

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	app "github.com/kiLeo13/SaladaFun/discord/padinho/internal/application/letterboxd"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/locale/ptbr"
)

// TestClassificationEmbeds ensures recognized Brazilian classifications prefix card ratings.
func TestClassificationEmbeds(t *testing.T) {
	for _, tc := range []struct {
		code, emoji string
	}{
		{app.ClassificationLivre, classificationLivreEmoji},
		{app.Classification6, classification6Emoji},
		{app.Classification10, classification10Emoji},
		{app.Classification12, classification12Emoji},
		{app.Classification14, classification14Emoji},
		{app.Classification16, classification16Emoji},
		{app.Classification18, classification18Emoji},
		{"", ""},
		{"unknown", ""},
	} {
		for _, kind := range []string{"movie", "tv"} {
			title := &entity.LetterboxdTitle{ID: 1, TMDBID: 1, OriginalTitle: "Title", MediaType: kind, ReleaseYear: new(2020), ContentRatings: []entity.ContentRating{{Country: "BR", Certification: tc.code}}}
			card := titleCard(title, nil, kind+":1")
			checkPayload(t, card)
			wantFooter := ptbr.LetterboxdNoRatings
			if tc.emoji != "" {
				wantFooter = tc.emoji + " " + wantFooter
			}
			if got := card.Embeds[0].Footer.Text; got != wantFooter {
				t.Fatalf("%s %s footer: got %q, want %q", kind, tc.code, got, wantFooter)
			}
			if len(card.Embeds[0].Fields) != 1 {
				t.Fatal("unexpected extra metadata field")
			}
			for _, field := range card.Embeds[0].Fields {
				if field.Name == ptbr.LetterboxdAgeRating {
					t.Fatal("classification field still visible")
				}
			}
			payload, err := json.Marshal(card)
			if err != nil {
				t.Fatal(err)
			}
			if got := strings.Contains(string(payload), "class_"); got != (tc.emoji != "") {
				t.Fatalf("%s %s emoji visibility: %t", kind, tc.code, got)
			}
			if len(title.ContentRatings) != 1 || title.ContentRatings[0].Certification != tc.code {
				t.Fatal("presentation mutated cached classifications")
			}
		}
	}
}

// TestClassificationFooterRating keeps the classification and calculated rating separated by one space.
func TestClassificationFooterRating(t *testing.T) {
	title := &entity.LetterboxdTitle{
		ID:            1,
		TMDBID:        1,
		OriginalTitle: "Title",
		MediaType:     "movie",
		ReleaseYear:   new(2020),
		ContentRatings: []entity.ContentRating{{
			Country:       app.ClassificationCountryBR,
			Certification: app.Classification12,
		}},
	}

	card := titleCard(title, []app.Opinion{{Rating: 5}}, "movie:1")
	if got, want := card.Embeds[0].Footer.Text, classification12Emoji+" "+fmt.Sprintf(ptbr.LetterboxdRatingSingle, "5"); got != want {
		t.Fatalf("footer: got %q, want %q", got, want)
	}
}
