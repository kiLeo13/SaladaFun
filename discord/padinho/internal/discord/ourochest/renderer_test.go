package ourochest

import (
	"strings"
	"testing"

	appourochest "github.com/kiLeo13/SaladaFun/discord/padinho/internal/application/ourochest"
)

func TestRenderRecommendations(t *testing.T) {
	content := renderRecommendations(appourochest.Result{
		CandidateCount: 8,
		Recommendations: []appourochest.Recommendation{
			{Kind: appourochest.Balanced, Position: 16, RedProbability: 0.125, ImmediateValue: 41.25, InformationGain: 1.5, ExpectedCandidates: 3.25},
			{Kind: appourochest.Information, Position: 3, InformationGain: 2.25, ExpectedCandidates: 2.5},
		},
	})
	for _, expected := range []string{"botão 17", "linha 4, coluna 2", "12,5%", "41,2", "Mais informação", "8 posição"} {
		if !strings.Contains(content, expected) {
			t.Fatalf("content %q does not contain %q", content, expected)
		}
	}
}

func TestRenderNoSuggestion(t *testing.T) {
	if content := renderRecommendations(appourochest.Result{}); !strings.Contains(content, "Não há") {
		t.Fatalf("content = %q", content)
	}
}
