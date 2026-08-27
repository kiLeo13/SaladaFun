package ourochest

import (
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
	appourochest "github.com/kiLeo13/SaladaFun/discord/padinho/internal/application/ourochest"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/discord/theme"
)

func TestRenderRecommendations(t *testing.T) {
	components := renderRecommendations(appourochest.Result{
		CandidateCount: 8,
		Recommendations: []appourochest.Recommendation{
			{Kind: appourochest.Balanced, Position: 16, RedProbability: 0.125, ImmediateValue: 41.25, InformationGain: 1.5, ExpectedCandidates: 3.25},
			{Kind: appourochest.Information, Position: 3, InformationGain: 2.25, ExpectedCandidates: 2.5},
		},
	})
	if len(components) != 1 {
		t.Fatalf("components = %#v", components)
	}
	container, ok := components[0].(discordgo.Container)
	if !ok || container.AccentColor == nil || *container.AccentColor != theme.AccentColor {
		t.Fatalf("container = %#v", components[0])
	}
	if len(container.Components) != 4 {
		t.Fatalf("container components = %#v", container.Components)
	}
	content := componentText(components)
	for _, expected := range []string{"botão 17", "Linha 4, coluna 2", "12,5%", "41,2", "!toggleochelper"} {
		if !strings.Contains(content, expected) {
			t.Fatalf("content %q does not contain %q", content, expected)
		}
	}
	for _, excluded := range []string{"Mais informação", "botão 4", "8 posição"} {
		if strings.Contains(content, excluded) {
			t.Fatalf("content %q unexpectedly contains %q", content, excluded)
		}
	}
	separator, ok := container.Components[2].(discordgo.Separator)
	if !ok || separator.Divider == nil || !*separator.Divider {
		t.Fatalf("separator = %#v", container.Components[2])
	}
}

func TestRenderNoSuggestion(t *testing.T) {
	if content := componentText(renderRecommendations(appourochest.Result{})); !strings.Contains(content, "Não há") {
		t.Fatalf("content = %q", content)
	}
}

// componentText flattens text displays nested in helper containers for assertions.
func componentText(components []discordgo.MessageComponent) string {
	var content strings.Builder
	for _, component := range components {
		switch value := component.(type) {
		case discordgo.TextDisplay:
			content.WriteString(value.Content)
			content.WriteByte('\n')
		case discordgo.Container:
			content.WriteString(componentText(value.Components))
		}
	}
	return content.String()
}
