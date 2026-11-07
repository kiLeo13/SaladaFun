package letterboxd

import (
	"testing"

	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/locale/ptbr"
)

// TestClassificationEmbeds verifies exact application emoji IDs in real title cards.
func TestClassificationEmbeds(t *testing.T) {
	for _, tc := range []struct{ code, want string }{
		{"L", "<:class_l:1568484877429374997> Livre"},
		{"6", "<:class_6:1568484865962414120> 6 anos"},
		{"10", "<:class_10:1568484867866628116> 10 anos"},
		{"12", "<:class_12:1568484870001393776> 12 anos"},
		{"14", "<:class_14:1568484871448428604> 14 anos"},
		{"16", "<:class_16:1568484873033744436> 16 anos"},
		{"18", "<:class_18:1568484874862592030> 18 anos"},
		{"", "Não informada"}, {"unknown", "Não informada"},
	} {
		for _, kind := range []string{"movie", "tv"} {
			title := &entity.LetterboxdTitle{ID: 1, TMDBID: 1, OriginalTitle: "Title", MediaType: kind, ReleaseYear: 2020, ContentRatings: []entity.ContentRating{{Country: "BR", Certification: tc.code}}}
			card := titleCard(title, nil, kind+":1")
			checkPayload(t, card)
			value := ""
			for _, field := range card.Embeds[0].Fields {
				if field.Name == ptbr.LetterboxdAgeRating {
					value = field.Value
				}
			}
			if value != tc.want {
				t.Fatalf("%s/%s field = %q, want %q", kind, tc.code, value, tc.want)
			}
		}
	}
	if got := classificationText([]entity.ContentRating{{Country: "US", Certification: "18"}}); got != ptbr.LetterboxdAgeUnknown {
		t.Fatal("foreign classification used as BR")
	}
}
