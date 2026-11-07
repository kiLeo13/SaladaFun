package letterboxd

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/locale/ptbr"
)

// TestClassificationEmbeds ensures classifications stay hidden even when metadata is populated.
func TestClassificationEmbeds(t *testing.T) {
	for _, code := range []string{"L", "6", "10", "12", "14", "16", "18", "", "unknown"} {
		for _, kind := range []string{"movie", "tv"} {
			title := &entity.LetterboxdTitle{ID: 1, TMDBID: 1, OriginalTitle: "Title", MediaType: kind, ReleaseYear: new(2020), ContentRatings: []entity.ContentRating{{Country: "BR", Certification: code}}}
			card := titleCard(title, nil, kind+":1")
			checkPayload(t, card)
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
			if strings.Contains(string(payload), "class_") {
				t.Fatal("classification emoji still visible")
			}
			if len(title.ContentRatings) != 1 || title.ContentRatings[0].Certification != code {
				t.Fatal("presentation mutated cached classifications")
			}
		}
	}
}
