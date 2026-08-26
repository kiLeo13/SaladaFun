package ourochest

import (
	"fmt"
	"strings"

	appourochest "github.com/kiLeo13/SaladaFun/discord/padinho/internal/application/ourochest"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/locale/ptbr"
)

// renderRecommendations produces a compact explanation for every distinct objective winner.
func renderRecommendations(result appourochest.Result) string {
	if len(result.Recommendations) == 0 {
		return ptbr.OuroChestNoSuggestion
	}
	var content strings.Builder
	for index, recommendation := range result.Recommendations {
		if index > 0 {
			content.WriteString("\n\n")
		}
		row := recommendation.Position/appourochest.BoardWidth + 1
		column := recommendation.Position%appourochest.BoardWidth + 1
		content.WriteString(fmt.Sprintf(recommendationTitle(recommendation.Kind), recommendation.Position+1, row, column))
		content.WriteByte('\n')
		content.WriteString(recommendationReason(recommendation))
	}
	content.WriteString(fmt.Sprintf(ptbr.OuroChestCandidateFooter, result.CandidateCount))
	return content.String()
}

// recommendationTitle returns the localized heading for one strategy objective.
func recommendationTitle(kind appourochest.RecommendationKind) string {
	switch kind {
	case appourochest.Information:
		return ptbr.OuroChestInformationTitle
	case appourochest.Reward:
		return ptbr.OuroChestRewardTitle
	case appourochest.DirectRed:
		return ptbr.OuroChestRedTitle
	default:
		return ptbr.OuroChestBalancedTitle
	}
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
