package letterboxd

import (
	"fmt"

	app "github.com/kiLeo13/SaladaFun/discord/padinho/internal/application/letterboxd"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/locale/ptbr"
)

// Application-owned classification emojis use Discord markup for embed text.
// Keep these presentation identifiers outside the application/domain packages.
const (
	classificationLivreEmoji = "<:class_l:1568484877429374997>"
	classification6Emoji     = "<:class_6:1568484865962414120>"
	classification10Emoji    = "<:class_10:1568484867866628116>"
	classification12Emoji    = "<:class_12:1568484870001393776>"
	classification14Emoji    = "<:class_14:1568484871448428604>"
	classification16Emoji    = "<:class_16:1568484873033744436>"
	classification18Emoji    = "<:class_18:1568484874862592030>"
)

// classificationText renders recognized Brazilian codes with application emojis.
func classificationText(ratings []entity.ContentRating) string {
	code := app.BrazilianClassification(ratings)
	emoji := ""
	switch code {
	case app.ClassificationLivre:
		return classificationLivreEmoji + " " + ptbr.LetterboxdAgeLivre
	case app.Classification6:
		emoji = classification6Emoji
	case app.Classification10:
		emoji = classification10Emoji
	case app.Classification12:
		emoji = classification12Emoji
	case app.Classification14:
		emoji = classification14Emoji
	case app.Classification16:
		emoji = classification16Emoji
	case app.Classification18:
		emoji = classification18Emoji
	default:
		return ptbr.LetterboxdAgeUnknown
	}
	return emoji + " " + fmt.Sprintf(ptbr.LetterboxdAgeYears, code)
}
