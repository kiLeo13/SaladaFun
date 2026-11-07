package letterboxd

import (
	"testing"

	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
)

// TestBrazilianClassification covers every supported code and conservative selection.
func TestBrazilianClassification(t *testing.T) {
	for _, code := range []string{ClassificationLivre, Classification6, Classification10, Classification12, Classification14, Classification16, Classification18} {
		if got := BrazilianClassification([]entity.ContentRating{{Country: " br ", Certification: " " + code + " "}}); got != code {
			t.Fatalf("%s became %s", code, got)
		}
	}
	for _, ratings := range [][]entity.ContentRating{nil, {}, {{Country: "US", Certification: "18"}}, {{Country: "BR", Certification: ""}}, {{Country: "BR", Certification: "unknown"}}} {
		if got := BrazilianClassification(ratings); got != "" {
			t.Fatalf("unknown became %q", got)
		}
	}
	ratings := []entity.ContentRating{{Country: "BR", Certification: "12", ReleaseType: 3}, {Country: "BR", Certification: "16", ReleaseType: 4}, {Country: "BR", Certification: "L"}, {Country: "US", Certification: "18"}, {Country: "BR", Certification: "unknown"}}
	if got := BrazilianClassification(ratings); got != Classification16 {
		t.Fatalf("conflicting release ratings: %s", got)
	}
}
