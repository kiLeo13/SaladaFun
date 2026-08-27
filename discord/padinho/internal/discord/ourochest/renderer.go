package ourochest

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	appourochest "github.com/kiLeo13/SaladaFun/discord/padinho/internal/application/ourochest"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/discord/theme"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/locale/ptbr"
)

// renderRecommendations produces a Components V2 container for the top-ranked move.
func renderRecommendations(result appourochest.Result) []discordgo.MessageComponent {
	if len(result.Recommendations) == 0 {
		return renderStatus(ptbr.OuroChestNoSuggestion)
	}
	recommendation := result.Recommendations[0]
	row := recommendation.Position/appourochest.BoardWidth + 1
	column := recommendation.Position%appourochest.BoardWidth + 1
	return helperContainer(
		fmt.Sprintf(ptbr.OuroChestRecommendedTitle, recommendation.Position+1),
		fmt.Sprintf(ptbr.OuroChestRecommendedSubtitle, row, column, recommendationReason(recommendation)),
	)
}

// renderStatus presents a non-recommendation helper state with the same visual treatment.
func renderStatus(content string) []discordgo.MessageComponent {
	return helperContainer(content, "")
}

// helperContainer builds the shared helper layout and automatic-assistance hint.
func helperContainer(title, subtitle string) []discordgo.MessageComponent {
	divider := true
	components := []discordgo.MessageComponent{discordgo.TextDisplay{Content: title}}
	if subtitle != "" {
		components = append(components, discordgo.TextDisplay{Content: subtitle})
	}
	components = append(components,
		discordgo.Separator{Divider: &divider},
		discordgo.TextDisplay{Content: ptbr.OuroChestDisableHint},
	)
	accent := theme.AccentColor
	return []discordgo.MessageComponent{discordgo.Container{AccentColor: &accent, Components: components}}
}

// recommendationReason explains the metric that made one button useful.
func recommendationReason(recommendation appourochest.Recommendation) string {
	red := formatDecimal(100 * recommendation.RedProbability)
	value := formatDecimal(recommendation.ImmediateValue)
	information := formatDecimal(recommendation.InformationGain)
	candidates := formatDecimal(recommendation.ExpectedCandidates)
	switch recommendation.Kind {
	case appourochest.Information:
		return fmt.Sprintf(ptbr.OuroChestInformationReason, information, candidates)
	case appourochest.Reward:
		return fmt.Sprintf(ptbr.OuroChestRewardReason, value, red)
	case appourochest.DirectRed:
		return fmt.Sprintf(ptbr.OuroChestRedReason, red, information)
	default:
		return fmt.Sprintf(ptbr.OuroChestBalancedReason, red, information, value)
	}
}

// formatDecimal renders one decimal using Brazilian punctuation.
func formatDecimal(value float64) string {
	return strings.ReplaceAll(fmt.Sprintf("%.1f", value), ".", ",")
}
