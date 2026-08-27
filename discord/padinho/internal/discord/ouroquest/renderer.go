package ouroquest

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	appouroquest "github.com/kiLeo13/SaladaFun/discord/padinho/internal/application/ouroquest"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/discord/theme"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/locale/ptbr"
)

// renderRecommendation builds one concise Components V2 recommendation.
func renderRecommendation(result appouroquest.Result) []discordgo.MessageComponent {
	if result.Recommendation == nil {
		return renderStatus(ptbr.OuroQuestNoSuggestion)
	}
	recommendation := result.Recommendation
	row := recommendation.Position/appouroquest.BoardWidth + 1
	column := recommendation.Position%appouroquest.BoardWidth + 1
	subtitle := fmt.Sprintf(ptbr.OuroQuestRecommendedSubtitle, row, column, formatDecimal(100*recommendation.PurpleProbability), formatDecimal(recommendation.ImmediateValue))
	return helperContainer(fmt.Sprintf(ptbr.OuroQuestRecommendedTitle, recommendation.Position+1), subtitle)
}

// renderStatus presents a helper state with the same visual treatment.
func renderStatus(content string) []discordgo.MessageComponent { return helperContainer(content, "") }

// helperContainer builds the shared Ouroquest layout.
func helperContainer(title, subtitle string) []discordgo.MessageComponent {
	divider := true
	children := []discordgo.MessageComponent{discordgo.TextDisplay{Content: title}}
	if subtitle != "" {
		children = append(children, discordgo.TextDisplay{Content: subtitle})
	}
	children = append(children, discordgo.Separator{Divider: &divider}, discordgo.TextDisplay{Content: ptbr.OuroQuestDisableHint})
	accent := theme.AccentColor
	return []discordgo.MessageComponent{discordgo.Container{AccentColor: &accent, Components: children}}
}

// formatDecimal renders one decimal using Brazilian punctuation.
func formatDecimal(value float64) string {
	return strings.ReplaceAll(fmt.Sprintf("%.1f", value), ".", ",")
}
