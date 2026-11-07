package letterboxd

import (
	"testing"

	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
)

// TestReleaseMetadata covers missing, planned, released, canceled and TV-specific states.
func TestReleaseMetadata(t *testing.T) {
	for _, tc := range []struct {
		kind, status string
		year         *int
		seasons      int
		name, value  string
	}{
		{"movie", "Released", new(2026), 0, "Lançamento", "2026"},
		{"movie", "Released", nil, 0, "Lançamento", "Desconhecido"},
		{"movie", "Planned", nil, 0, "Previsão de lançamento", "Sem data anunciada"},
		{"movie", "In Production", new(2027), 0, "Previsão de lançamento", "2027"},
		{"movie", "Post Production", nil, 0, "Previsão de lançamento", "Sem data anunciada"},
		{"movie", "Rumored", nil, 0, "Previsão de lançamento", "Sem data anunciada"},
		{"movie", "Canceled", nil, 0, "Lançamento", "Cancelado"},
		{"movie", "Canceled", new(2020), 0, "Lançamento", "2020"},
		{"movie", "", nil, 0, "Lançamento", "Desconhecido"},
		{"movie", "New Provider Status", nil, 0, "Lançamento", "Desconhecido"},
		{"tv", "Planned", nil, 0, "Previsão de lançamento", "Sem data anunciada"},
		{"tv", "Pilot", nil, 1, "Previsão de lançamento", "Sem data anunciada - 1 temporada"},
		{"tv", "In Production", new(2028), 2, "Previsão de lançamento", "2028 - 2 temporadas"},
		{"tv", "Returning Series", nil, 3, "Lançamento", "Desconhecido - 3 temporadas"},
		{"tv", "Ended", new(2018), 5, "Lançamento", "2018 - 5 temporadas"},
		{"tv", "Canceled", new(2019), 1, "Lançamento", "2019 - 1 temporada"},
		{"tv", "Canceled", nil, 0, "Lançamento", "Cancelado"},
		{"tv", "", nil, 0, "Lançamento", "Desconhecido"},
	} {
		t.Run(tc.kind+"/"+tc.status+"/"+tc.value, func(t *testing.T) {
			title := &entity.LetterboxdTitle{MediaType: tc.kind, Status: new(tc.status), ReleaseYear: tc.year, SeasonCount: tc.seasons, OriginalTitle: "Title", TMDBID: 1}
			field := titleCard(title, nil, tc.kind+":1").Embeds[0].Fields[0]
			if field.Name != tc.name || field.Value != tc.value || !field.Inline {
				t.Fatalf("release field: %#v", field)
			}
		})
	}
	if got := releaseField(&entity.LetterboxdTitle{}); got.Value != "Desconhecido" {
		t.Fatalf("nil status: %#v", got)
	}
}
