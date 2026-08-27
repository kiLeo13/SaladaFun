package ouroharvest

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	appouroharvest "github.com/kiLeo13/SaladaFun/discord/padinho/internal/application/ouroharvest"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/discord/theme"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/locale/ptbr"
)

// renderRecommendation builds the single Components V2 policy recommendation.
func renderRecommendation(position int, recommendation appouroharvest.Recommendation) []discordgo.MessageComponent {
	row := position/appouroharvest.BoardWidth + 1
	column := position%appouroharvest.BoardWidth + 1
	detail := fmt.Sprintf(
		ptbr.OuroHarvestRecommendedSubtitle,
		row, column, colorLabel(appouroharvest.Color(recommendation.Action)),
		formatDecimal(recommendation.ExpectedSP), formatDecimal(recommendation.AdvantageSP),
	)
	if recommendation.ChestProbability > 0 {
		detail += fmt.Sprintf(ptbr.OuroHarvestChestChance, formatDecimal(100*recommendation.ChestProbability))
	}
	return helperContainer(fmt.Sprintf(ptbr.OuroHarvestRecommendedTitle, position+1), detail)
}

// renderPurple builds the unconditional free-purple recommendation.
func renderPurple(position int, clicksLeft uint8) []discordgo.MessageComponent {
	row := position/appouroharvest.BoardWidth + 1
	column := position%appouroharvest.BoardWidth + 1
	return helperContainer(
		fmt.Sprintf(ptbr.OuroHarvestRecommendedTitle, position+1),
		fmt.Sprintf(ptbr.OuroHarvestPurpleSubtitle, row, column, clicksLeft),
	)
}

// renderStatus presents a non-recommendation state using the helper layout.
func renderStatus(content string) []discordgo.MessageComponent { return helperContainer(content, "") }

// helperContainer creates the accented helper and its automatic-assistance hint.
func helperContainer(title, subtitle string) []discordgo.MessageComponent {
	divider := true
	children := []discordgo.MessageComponent{discordgo.TextDisplay{Content: title}}
	if subtitle != "" {
		children = append(children, discordgo.TextDisplay{Content: subtitle})
	}
	children = append(children, discordgo.Separator{Divider: &divider}, discordgo.TextDisplay{Content: ptbr.OuroHarvestDisableHint})
	accent := theme.AccentColor
	return []discordgo.MessageComponent{discordgo.Container{AccentColor: &accent, Components: children}}
}

// colorLabel returns localized copy for one recommended sphere type.
func colorLabel(color appouroharvest.Color) string {
	labels := map[appouroharvest.Color]string{
		appouroharvest.Covered: ptbr.OuroHarvestColorCovered,
		appouroharvest.Blue:    ptbr.OuroHarvestColorBlue, appouroharvest.Teal: ptbr.OuroHarvestColorTeal,
		appouroharvest.Green: ptbr.OuroHarvestColorGreen, appouroharvest.Yellow: ptbr.OuroHarvestColorYellow,
		appouroharvest.Orange: ptbr.OuroHarvestColorOrange, appouroharvest.Red: ptbr.OuroHarvestColorRed,
		appouroharvest.Dark: ptbr.OuroHarvestColorDark, appouroharvest.Light: ptbr.OuroHarvestColorLight,
		appouroharvest.White: ptbr.OuroHarvestColorWhite,
	}
	return labels[color]
}

// formatDecimal renders one decimal using Brazilian punctuation.
func formatDecimal(value float64) string {
	return strings.ReplaceAll(fmt.Sprintf("%.1f", value), ".", ",")
}
