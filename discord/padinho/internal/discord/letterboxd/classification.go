package letterboxd

import app "github.com/kiLeo13/SaladaFun/discord/padinho/internal/application/letterboxd"

// Classification emojis belong to the Discord presentation layer.
const (
	classificationLivreEmoji = "<:class_l:1568484877429374997>"
	classification6Emoji     = "<:class_6:1568484865962414120>"
	classification10Emoji    = "<:class_10:1568484867866628116>"
	classification12Emoji    = "<:class_12:1568484870001393776>"
	classification14Emoji    = "<:class_14:1568484871448428604>"
	classification16Emoji    = "<:class_16:1568484873033744436>"
	classification18Emoji    = "<:class_18:1568484874862592030>"
)

// classificationEmoji returns the emoji assigned to a recognized Brazilian
// classification code, leaving unknown or missing classifications unrendered.
func classificationEmoji(code string) string {
	switch code {
	case app.ClassificationLivre:
		return classificationLivreEmoji
	case app.Classification6:
		return classification6Emoji
	case app.Classification10:
		return classification10Emoji
	case app.Classification12:
		return classification12Emoji
	case app.Classification14:
		return classification14Emoji
	case app.Classification16:
		return classification16Emoji
	case app.Classification18:
		return classification18Emoji
	default:
		return ""
	}
}
