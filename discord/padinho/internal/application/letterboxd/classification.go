package letterboxd

import (
	"strings"

	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
)

// Brazilian classification codes are provider values, not numeric ages.
// Missing or unrecognized codes must not be interpreted as Livre.
const (
	ClassificationCountryBR = "BR"
	ClassificationLivre     = "L"
	Classification6         = "6"
	Classification10        = "10"
	Classification12        = "12"
	Classification14        = "14"
	Classification16        = "16"
	Classification18        = "18"
)

// BrazilianClassification selects the highest recognized BR classification when
// release variants disagree. Foreign and unknown codes never imply Livre.
func BrazilianClassification(ratings []entity.ContentRating) string {
	selected := ""
	highest := -1
	for _, rating := range ratings {
		if !strings.EqualFold(strings.TrimSpace(rating.Country), ClassificationCountryBR) {
			continue
		}
		code := strings.ToUpper(strings.TrimSpace(rating.Certification))
		age := -1
		switch code {
		case ClassificationLivre:
			age = 0
		case Classification6:
			age = 6
		case Classification10:
			age = 10
		case Classification12:
			age = 12
		case Classification14:
			age = 14
		case Classification16:
			age = 16
		case Classification18:
			age = 18
		}
		if age > highest {
			selected, highest = code, age
		}
	}
	return selected
}
