package ouroquest

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
	appouroquest "github.com/kiLeo13/SaladaFun/discord/padinho/internal/application/ouroquest"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/discord/theme"
)

func TestListenerStartsOnlyAfterExactOQCorrelation(t *testing.T) {
	messenger := &testMessenger{sent: make(chan string, 1)}
	preferences := &testPreferences{automatic: true}
	listener, err := New("100", testEmojiIDs(), messenger, preferences, testLoader{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	listener.ctx = context.Background()
	t.Cleanup(func() {
		listener.mu.Lock()
		games := make([]*gameSession, 0, len(listener.games))
		for _, game := range listener.games {
			games = append(games, game)
		}
		listener.mu.Unlock()
		for _, game := range games {
			listener.stopGame(game.sourceID)
		}
	})
	board := &discordgo.Message{ID: "board", ChannelID: "channel", GuildID: "guild", Author: &discordgo.User{ID: "100", Bot: true}, Components: testRows(testButtons())}
	listener.handleMessageCreate(nil, &discordgo.MessageCreate{Message: board})
	select {
	case <-messenger.sent:
		t.Fatal("uncorrelated board started")
	default:
	}
	listener.handleMessageCreate(nil, &discordgo.MessageCreate{Message: &discordgo.Message{ID: "command", ChannelID: "channel", GuildID: "guild", Content: "$oq", Author: &discordgo.User{ID: "200"}}})
	listener.handleMessageCreate(nil, &discordgo.MessageCreate{Message: board})
	select {
	case content := <-messenger.sent:
		if !strings.Contains(content, "botão 7") {
			t.Fatalf("content = %q", content)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("helper was not sent")
	}
}

func TestCommandAndUnavailableRecognition(t *testing.T) {
	for _, command := range []string{"$oq", " $OQ ", "$ouroquest"} {
		if !isOQCommand(command) {
			t.Fatalf("command %q not recognized", command)
		}
	}
	for _, command := range []string{"$oc", "$oh", "please $oq", "$oq 2"} {
		if isOQCommand(command) {
			t.Fatalf("command %q recognized", command)
		}
	}
	message := &discordgo.Message{Content: "Você não tem $oq suficientes de hoje."}
	if !isUnavailable(message) {
		t.Fatal("unavailable response not recognized")
	}
}

func TestParseBoardMapsColorsAndPaidClicks(t *testing.T) {
	buttons := make([]discordgo.Button, appouroquest.CellCount)
	for position := range buttons {
		buttons[position] = discordgo.Button{Emoji: &discordgo.ComponentEmoji{ID: "unknown"}}
	}
	buttons[0].Emoji.ID = "101"
	buttons[0].Disabled = true
	buttons[1].Emoji.ID = "106"
	buttons[1].Disabled = true
	message := &discordgo.Message{Components: testRows(buttons)}
	snapshot, ok := parseBoard(message, map[string]appouroquest.Color{"101": appouroquest.Blue, "106": appouroquest.Purple})
	if !ok || snapshot.board[0] != appouroquest.Blue || snapshot.board[1] != appouroquest.Purple || snapshot.revealed != 2 || snapshot.terminal {
		t.Fatalf("snapshot = %#v, %t", snapshot, ok)
	}
	if _, ok := parseBoard(&discordgo.Message{}, nil); ok {
		t.Fatal("invalid grid accepted")
	}
}

func TestRenderRecommendationUsesOneAccentedContainer(t *testing.T) {
	components := renderRecommendation(appouroquest.Result{Recommendation: &appouroquest.Recommendation{Position: 6, PurpleProbability: .16, ImmediateValue: 48.611}})
	if len(components) != 1 {
		t.Fatalf("components = %#v", components)
	}
	container, ok := components[0].(discordgo.Container)
	if !ok || container.AccentColor == nil || *container.AccentColor != theme.AccentColor {
		t.Fatalf("container = %#v", components[0])
	}
	text := componentText(container.Components)
	for _, want := range []string{"botão 7", "Linha 2, coluna 2", "16,0%", "48,6", "!toggleoqhelper"} {
		if !strings.Contains(text, want) {
			t.Fatalf("text %q lacks %q", text, want)
		}
	}
	if !strings.Contains(componentText(renderRecommendation(appouroquest.Result{})), "Não há") {
		t.Fatal("missing empty status")
	}
}

func testRows(buttons []discordgo.Button) []discordgo.MessageComponent {
	rows := make([]discordgo.MessageComponent, 0, appouroquest.BoardWidth)
	for row := 0; row < appouroquest.BoardWidth; row++ {
		children := make([]discordgo.MessageComponent, 0, appouroquest.BoardWidth)
		for column := 0; column < appouroquest.BoardWidth; column++ {
			children = append(children, buttons[row*appouroquest.BoardWidth+column])
		}
		rows = append(rows, discordgo.ActionsRow{Components: children})
	}
	return rows
}
func componentText(components []discordgo.MessageComponent) string {
	var result strings.Builder
	for _, component := range components {
		switch value := component.(type) {
		case discordgo.TextDisplay:
			result.WriteString(value.Content)
		case discordgo.Container:
			result.WriteString(componentText(value.Components))
		}
	}
	return result.String()
}

func testButtons() []discordgo.Button {
	buttons := make([]discordgo.Button, appouroquest.CellCount)
	for position := range buttons {
		buttons[position] = discordgo.Button{Emoji: &discordgo.ComponentEmoji{ID: "999"}}
	}
	return buttons
}
func testEmojiIDs() EmojiIDs {
	return EmojiIDs{Blue: "101", Teal: "102", Green: "103", Yellow: "104", Orange: "105", Purple: "106", Red: "107"}
}

type testPreferences struct{ automatic bool }

func (p *testPreferences) AutoMudaeOQ(uint64) (bool, error) { return p.automatic, nil }
func (p *testPreferences) ToggleAutoMudaeOQ(uint64) (bool, error) {
	p.automatic = !p.automatic
	return p.automatic, nil
}

type testLoader struct{}

func (testLoader) LoadMessage(string, string) (*discordgo.Message, error) { return nil, nil }

type testMessenger struct {
	mu   sync.Mutex
	next int
	sent chan string
}

func (m *testMessenger) SendReply(_, _, _ string, components []discordgo.MessageComponent) (string, error) {
	m.mu.Lock()
	m.next++
	id := fmt.Sprintf("helper-%d", m.next)
	m.mu.Unlock()
	m.sent <- componentText(components)
	return id, nil
}
func (m *testMessenger) EditMessage(string, string, []discordgo.MessageComponent) error { return nil }
func (m *testMessenger) DeleteMessage(string, string) error                             { return nil }
