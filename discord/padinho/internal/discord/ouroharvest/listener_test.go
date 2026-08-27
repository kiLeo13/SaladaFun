package ouroharvest

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
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/command"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/messagecommand"
)

func TestListenerRequiresExactCommandAndStrongBoardSignature(t *testing.T) {
	messenger := &harvestMessenger{sent: make(chan string, 2)}
	listener := newTestListener(t, messenger, &harvestPreferences{automatic: true}, harvestLoader{})
	board := testHarvestMessage()
	listener.handleMessageCreate(nil, &discordgo.MessageCreate{Message: board})
	assertNoHarvestMessage(t, messenger.sent)
	listener.handleMessageCreate(nil, &discordgo.MessageCreate{Message: userMessage("command-oc", "$oc")})
	listener.handleMessageCreate(nil, &discordgo.MessageCreate{Message: board})
	assertNoHarvestMessage(t, messenger.sent)
	listener.handleMessageCreate(nil, &discordgo.MessageCreate{Message: userMessage("command-oh", "$oh")})
	weak := *board
	weak.Content = "generic 5x5 board"
	listener.handleMessageCreate(nil, &discordgo.MessageCreate{Message: &weak})
	assertNoHarvestMessage(t, messenger.sent)
	listener.handleMessageCreate(nil, &discordgo.MessageCreate{Message: board})
	select {
	case content := <-messenger.sent:
		if !strings.Contains(content, "botão 1") || !strings.Contains(content, "esfera branca") {
			t.Fatalf("content = %q", content)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("helper was not sent")
	}
}

func TestDisabledAutomationStillAllowsManualHelp(t *testing.T) {
	messenger := &harvestMessenger{sent: make(chan string, 2)}
	board := testHarvestMessage()
	preferences := &harvestPreferences{automatic: false}
	listener := newTestListener(t, messenger, preferences, harvestLoader{message: board})
	listener.handleMessageCreate(nil, &discordgo.MessageCreate{Message: userMessage("command-oh", "$oh")})
	listener.handleMessageCreate(nil, &discordgo.MessageCreate{Message: board})
	assertNoHarvestMessage(t, messenger.sent)
	request := &messagecommand.Request{
		Actor: command.Actor{UserID: "200"}, GuildID: "guild", ChannelID: "channel",
		ReplyToID: "board", Responder: &harvestResponder{},
	}
	if err := listener.handleHelperCommand(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	select {
	case <-messenger.sent:
	case <-time.After(2 * time.Second):
		t.Fatal("manual helper was not sent")
	}
}

func TestToggleCommandChangesOnlyAutomaticPreference(t *testing.T) {
	messenger := &harvestMessenger{sent: make(chan string, 1)}
	preferences := &harvestPreferences{automatic: true}
	listener := newTestListener(t, messenger, preferences, harvestLoader{})
	responder := &harvestResponder{}
	request := &messagecommand.Request{Actor: command.Actor{UserID: "200"}, Responder: responder}
	if err := listener.handleToggleCommand(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if preferences.automatic || !strings.Contains(responder.content, "desativada") || !strings.Contains(responder.content, "!ohhelper") {
		t.Fatalf("preference = %t, response = %q", preferences.automatic, responder.content)
	}
}

func TestTerminalUpdateDeletesHelper(t *testing.T) {
	messenger := &harvestMessenger{sent: make(chan string, 1), deleted: make(chan string, 1)}
	listener := newTestListener(t, messenger, &harvestPreferences{automatic: true}, harvestLoader{})
	listener.handleMessageCreate(nil, &discordgo.MessageCreate{Message: userMessage("command-oh", "$oh")})
	board := testHarvestMessage()
	listener.handleMessageCreate(nil, &discordgo.MessageCreate{Message: board})
	select {
	case <-messenger.sent:
	case <-time.After(2 * time.Second):
		t.Fatal("helper was not sent")
	}
	buttons := harvestButtons("111")
	for position := range buttons {
		buttons[position].Disabled = true
	}
	terminal := &discordgo.Message{ID: "board", ChannelID: "channel", GuildID: "guild", Components: harvestRows(buttons)}
	listener.handleMessageUpdate(nil, &discordgo.MessageUpdate{Message: terminal})
	select {
	case <-messenger.deleted:
	case <-time.After(2 * time.Second):
		t.Fatal("helper was not deleted")
	}
}

func newTestListener(t *testing.T, messenger *harvestMessenger, preferences *harvestPreferences, loader harvestLoader) *Listener {
	t.Helper()
	listener, err := New("100", testHarvestEmojiIDs(), messenger, preferences, loader, slog.New(slog.NewTextHandler(io.Discard, nil)))
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
	return listener
}

func testHarvestEmojiIDs() EmojiIDs {
	return EmojiIDs{Covered: "101", Blue: "102", Teal: "103", Green: "104", Yellow: "105", Orange: "106", Red: "107", Purple: "108", Dark: "109", Light: "110", White: "111"}
}

func testHarvestMessage() *discordgo.Message {
	return &discordgo.Message{
		ID: "board", ChannelID: "channel", GuildID: "guild", Author: &discordgo.User{ID: "100", Bot: true},
		Content:    "Você pode clicar **5** vezes nos botões abaixo (por 2 minutos. Só você pode clicar). Esferas azuis revelam 3 botões; esferas cianas revelam 1.",
		Components: harvestRows(harvestButtons("111")),
	}
}

func userMessage(id, content string) *discordgo.Message {
	return &discordgo.Message{ID: id, ChannelID: "channel", GuildID: "guild", Content: content, Author: &discordgo.User{ID: "200"}}
}

func assertNoHarvestMessage(t *testing.T, messages <-chan string) {
	t.Helper()
	select {
	case message := <-messages:
		t.Fatalf("unexpected helper message %q", message)
	case <-time.After(25 * time.Millisecond):
	}
}

type harvestPreferences struct{ automatic bool }

func (p *harvestPreferences) AutoMudaeOH(uint64) (bool, error) { return p.automatic, nil }
func (p *harvestPreferences) ToggleAutoMudaeOH(uint64) (bool, error) {
	p.automatic = !p.automatic
	return p.automatic, nil
}

type harvestLoader struct{ message *discordgo.Message }

func (l harvestLoader) LoadMessage(string, string) (*discordgo.Message, error) { return l.message, nil }

type harvestResponder struct{ content string }

func (r *harvestResponder) Reply(content string) error { r.content = content; return nil }

type harvestMessenger struct {
	mu      sync.Mutex
	next    int
	sent    chan string
	deleted chan string
}

func (m *harvestMessenger) SendReply(_, _, _ string, components []discordgo.MessageComponent) (string, error) {
	m.mu.Lock()
	m.next++
	id := fmt.Sprintf("helper-%d", m.next)
	m.mu.Unlock()
	m.sent <- harvestComponentText(components)
	return id, nil
}

func (m *harvestMessenger) EditMessage(string, string, []discordgo.MessageComponent) error {
	return nil
}
func (m *harvestMessenger) DeleteMessage(_, messageID string) error {
	if m.deleted != nil {
		m.deleted <- messageID
	}
	return nil
}
