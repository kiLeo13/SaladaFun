package ouroharvest

import (
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
	appouroharvest "github.com/kiLeo13/SaladaFun/discord/padinho/internal/application/ouroharvest"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/discord/theme"
)

func TestRecognizesOuroharvestInstructions(t *testing.T) {
	message := &discordgo.Message{Content: "Você pode clicar **5** vezes nos botões abaixo (por 2 minutos. Só você pode clicar). Os botões de esfera têm valores diferentes dependendo da cor, como kakera. Esferas azuis revelam 3 botões; esferas cianas revelam 1."}
	if !isOuroharvestBoard(message) {
		t.Fatal("Portuguese instructions were not recognized")
	}
	message.Content = "Você pode clicar 5 vezes, mas isto não é o jogo completo."
	if isOuroharvestBoard(message) {
		t.Fatal("weak instruction fragment was recognized")
	}
}

func TestParseBoardDerivesClicksRefundsAndChest(t *testing.T) {
	buttons := harvestButtons("covered")
	buttons[0] = button("blue", false)
	buttons[1] = button("purple", false)
	buttons[2] = button("dark", true)
	buttons[3] = button("covered", true)
	message := &discordgo.Message{
		Content:    "<:spD:108> se transforma em <:spP:107>",
		Components: harvestRows(buttons),
	}
	snapshot, ok := parseBoard(message, harvestColors())
	if !ok {
		t.Fatal("valid board rejected")
	}
	if snapshot.state.ClicksLeft != 4 || snapshot.state.Blue != 1 || snapshot.state.Covered != 21 || !snapshot.state.ChestFound {
		t.Fatalf("state = %#v", snapshot.state)
	}
	if position, ok := snapshot.purplePosition(); !ok || position != 1 {
		t.Fatalf("purple position = %d, %t", position, ok)
	}
	if position, ok := snapshot.positionFor(appouroharvest.Action(appouroharvest.Blue)); !ok || position != 0 {
		t.Fatalf("blue position = %d, %t", position, ok)
	}
}

func TestParseBoardRejectsUnknownEmojiAndInvalidShape(t *testing.T) {
	buttons := harvestButtons("covered")
	buttons[4].Emoji.ID = "unconfigured"
	if _, ok := parseBoard(&discordgo.Message{Components: harvestRows(buttons)}, harvestColors()); ok {
		t.Fatal("unknown emoji accepted")
	}
	if _, ok := parseBoard(&discordgo.Message{}, harvestColors()); ok {
		t.Fatal("missing grid accepted")
	}
}

func TestRenderRecommendationAndPurpleUseAccentedContainer(t *testing.T) {
	components := renderRecommendation(6, appouroharvest.Recommendation{
		Action: appouroharvest.Action(appouroharvest.Covered), ExpectedSP: 123.45,
		AdvantageSP: 8.76, ChestProbability: .01998,
	})
	if len(components) != 1 {
		t.Fatalf("components = %#v", components)
	}
	container, ok := components[0].(discordgo.Container)
	if !ok || container.AccentColor == nil || *container.AccentColor != theme.AccentColor {
		t.Fatalf("container = %#v", components[0])
	}
	text := harvestComponentText(container.Components)
	for _, want := range []string{"botão 7", "Linha 2, coluna 2", "botão coberto", "123,5", "8,8", "2,0%", "!toggleohhelper"} {
		if !strings.Contains(text, want) {
			t.Fatalf("text %q lacks %q", text, want)
		}
	}
	if text := harvestComponentText(renderPurple(0, 5)); !strings.Contains(text, "roxa gratuita") || !strings.Contains(text, "5 cliques") {
		t.Fatalf("purple text = %q", text)
	}
}

func button(emojiID string, disabled bool) discordgo.Button {
	return discordgo.Button{Emoji: &discordgo.ComponentEmoji{ID: emojiID}, Disabled: disabled}
}

func harvestButtons(emojiID string) []discordgo.Button {
	buttons := make([]discordgo.Button, appouroharvest.CellCount)
	for position := range buttons {
		buttons[position] = button(emojiID, false)
	}
	return buttons
}

func harvestRows(buttons []discordgo.Button) []discordgo.MessageComponent {
	rows := make([]discordgo.MessageComponent, 0, appouroharvest.BoardWidth)
	for row := 0; row < appouroharvest.BoardWidth; row++ {
		children := make([]discordgo.MessageComponent, 0, appouroharvest.BoardWidth)
		for column := 0; column < appouroharvest.BoardWidth; column++ {
			children = append(children, buttons[row*appouroharvest.BoardWidth+column])
		}
		rows = append(rows, discordgo.ActionsRow{Components: children})
	}
	return rows
}

func harvestColors() map[string]appouroharvest.Color {
	return map[string]appouroharvest.Color{
		"covered": appouroharvest.Covered, "blue": appouroharvest.Blue, "teal": appouroharvest.Teal,
		"green": appouroharvest.Green, "yellow": appouroharvest.Yellow, "orange": appouroharvest.Orange,
		"red": appouroharvest.Red, "purple": appouroharvest.Purple, "dark": appouroharvest.Dark,
		"light": appouroharvest.Light, "white": appouroharvest.White,
	}
}

func harvestComponentText(components []discordgo.MessageComponent) string {
	var result strings.Builder
	for _, component := range components {
		switch value := component.(type) {
		case discordgo.TextDisplay:
			result.WriteString(value.Content)
		case discordgo.Container:
			result.WriteString(harvestComponentText(value.Components))
		}
	}
	return result.String()
}
