package ptbr

import (
	"strings"
	"testing"
)

// TestLetterboxdHeadingCapitalization verifies the display spelling of multiword headings.
func TestLetterboxdHeadingCapitalization(t *testing.T) {
	for _, test := range []struct {
		name string
		got  string
		want string
	}{
		{"release", LetterboxdReleaseExpected, "Previsão de Lançamento"},
		{"age", LetterboxdAgeRating, "Faixa Etária"},
		{"rating", LetterboxdRating, "Nota Geral: %s/5 · avaliado %d vezes"},
		{"single rating", LetterboxdRatingSingle, "Nota Geral: %s/5 · avaliado 1 vez"},
		{"local mode", strings.SplitN(LetterboxdAboutLocal, "\n", 2)[0], "### ⚠️ Modo Local"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.got != test.want {
				t.Errorf("got %q, want %q", test.got, test.want)
			}
		})
	}
}
