package ourochest

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
)

func TestListenerCorrelatesOCAndPublishesUpdates(t *testing.T) {
	listener, messenger := testListener(t)
	listener.handleMessageCreate(nil, commandEvent("$oc", "300"))
	board := testBoardMessage("board", "", "grid-cell")
	listener.handleMessageCreate(nil, &discordgo.MessageCreate{Message: board})
	sent := receiveCall(t, messenger.sent)
	if sent.sourceID != "board" || !strings.Contains(sent.content, "Equilibrada") {
		t.Fatalf("sent = %#v", sent)
	}

	buttons, _ := flattenButtons(board.Components)
	buttons[16].Emoji = &discordgo.ComponentEmoji{ID: "101"}
	buttons[16].Disabled = true
	updated := *board
	updated.Components = buttonRows(buttons)
	listener.handleMessageUpdate(nil, &discordgo.MessageUpdate{Message: &updated})
	edited := receiveCall(t, messenger.edited)
	if edited.messageID != sent.messageID || edited.content == sent.content {
		t.Fatalf("edited = %#v", edited)
	}

	for position, id := range []string{"101", "102", "103", "104", "105"} {
		buttons[position].Emoji = &discordgo.ComponentEmoji{ID: id}
		buttons[position].Disabled = true
	}
	updated.Components = buttonRows(buttons)
	listener.handleMessageUpdate(nil, &discordgo.MessageUpdate{Message: &updated})
	deleted := receiveCall(t, messenger.deleted)
	if deleted.messageID != sent.messageID {
		t.Fatalf("deleted = %#v", deleted)
	}
}

func TestListenerDoesNotAutomaticallyStartUncorrelatedOC(t *testing.T) {
	listener, _ := testListener(t)
	board := testBoardMessage("board", "$oc Ourochest", "oc:cell")
	listener.handleMessageCreate(nil, &discordgo.MessageCreate{Message: board})
	listener.mu.Lock()
	defer listener.mu.Unlock()
	if len(listener.games) != 0 || listener.boards[board.ID] != gameOC {
		t.Fatalf("games = %d, board kind = %d", len(listener.games), listener.boards[board.ID])
	}
}

func TestListenerNeverGuessesAmbiguousOrOHBoards(t *testing.T) {
	listener, _ := testListener(t)
	ambiguous := testBoardMessage("ambiguous", "", "grid-cell")
	listener.handleMessageCreate(nil, &discordgo.MessageCreate{Message: ambiguous})
	listener.handleMessageCreate(nil, commandEvent("$oh", "300"))
	oh := testBoardMessage("oh", "$oh sphere harvest", "grid-cell")
	listener.handleMessageCreate(nil, &discordgo.MessageCreate{Message: oh})
	listener.mu.Lock()
	games := len(listener.games)
	listener.mu.Unlock()
	if games != 0 {
		t.Fatalf("games = %d", games)
	}
}

func TestListenerKeepsCorrelationAcrossDifferentExplicitGame(t *testing.T) {
	listener, messenger := testListener(t)
	listener.handleMessageCreate(nil, commandEvent("$oc", "300"))
	listener.handleMessageCreate(nil, &discordgo.MessageCreate{Message: testBoardMessage("other-oh", "$oh sphere harvest", "oh:cell")})
	listener.handleMessageCreate(nil, &discordgo.MessageCreate{Message: testBoardMessage("wanted-oc", "", "grid-cell")})
	call := receiveCall(t, messenger.sent)
	if call.sourceID != "wanted-oc" {
		t.Fatalf("source ID = %q", call.sourceID)
	}
}

func TestListenerCancelsFailedOCBeforeImmediateOHBoard(t *testing.T) {
	tests := []string{
		"Você não tem $oc suficientes de hoje. Tempo de espera até o recarregamento: 7h 10 min.",
		"Você não tem nenhum $oc disponível! Você precisa da Emblema de kakera **Diamante lvl IV** ou premium",
	}
	for index, failure := range tests {
		t.Run(fmt.Sprint(index), func(t *testing.T) {
			listener, _ := testListener(t)
			listener.handleMessageCreate(nil, commandEvent("$oc", "300"))
			listener.handleMessageCreate(nil, &discordgo.MessageCreate{Message: &discordgo.Message{
				ID: "failure", ChannelID: "300", GuildID: "400", Content: failure,
				Author: &discordgo.User{ID: "100", Bot: true},
			}})
			listener.handleMessageCreate(nil, commandEvent("$oh", "300"))
			listener.handleMessageCreate(nil, &discordgo.MessageCreate{Message: testBoardMessage("oh", "", "grid-cell")})
			listener.mu.Lock()
			defer listener.mu.Unlock()
			if len(listener.games) != 0 || len(listener.pending) != 0 || listener.boards["oh"] != gameOH {
				t.Fatalf("games = %d, pending = %d, board = %d", len(listener.games), len(listener.pending), listener.boards["oh"])
			}
		})
	}
}

func TestListenerUsesMostRecentCommandWhenCooldownEventIsDelayed(t *testing.T) {
	listener, _ := testListener(t)
	listener.handleMessageCreate(nil, commandEvent("$oc", "300"))
	listener.handleMessageCreate(nil, commandEvent("$oh", "300"))
	listener.handleMessageCreate(nil, &discordgo.MessageCreate{Message: testBoardMessage("oh", "", "grid-cell")})
	listener.handleMessageCreate(nil, &discordgo.MessageCreate{Message: &discordgo.Message{
		ID: "failure", ChannelID: "300", GuildID: "400",
		Content: "Você não tem $oc suficientes de hoje. Tempo de espera até o recarregamento: 7h 10 min.",
		Author:  &discordgo.User{ID: "100", Bot: true},
	}})
	listener.mu.Lock()
	defer listener.mu.Unlock()
	if len(listener.games) != 0 || len(listener.pending) != 0 || listener.boards["oh"] != gameOH {
		t.Fatalf("games = %d, pending = %d, board = %d", len(listener.games), len(listener.pending), listener.boards["oh"])
	}
}

func TestListenerPrefersExactMudaeMessageReference(t *testing.T) {
	listener, messenger := testListener(t)
	listener.handleMessageCreate(nil, commandEvent("$oc", "300"))
	listener.handleMessageCreate(nil, commandEvent("$oh", "300"))
	board := testBoardMessage("oc", "", "grid-cell")
	board.MessageReference = &discordgo.MessageReference{MessageID: "command-$oc"}
	listener.handleMessageCreate(nil, &discordgo.MessageCreate{Message: board})
	if call := receiveCall(t, messenger.sent); call.sourceID != "oc" {
		t.Fatalf("helper = %#v", call)
	}
	listener.mu.Lock()
	defer listener.mu.Unlock()
	if len(listener.pending["300"]) != 1 || listener.pending["300"][0].kind != gameOH {
		t.Fatalf("pending = %#v", listener.pending["300"])
	}
}

func TestListenerHonorsDisabledAutomaticPreference(t *testing.T) {
	listener, _ := testListener(t)
	listener.preferences.(*fakePreferences).automatic = false
	listener.handleMessageCreate(nil, commandEvent("$oc", "300"))
	listener.handleMessageCreate(nil, &discordgo.MessageCreate{Message: testBoardMessage("board", "", "grid-cell")})
	listener.mu.Lock()
	defer listener.mu.Unlock()
	if len(listener.games) != 0 || listener.boards["board"] != gameOC {
		t.Fatalf("games = %d, board = %d", len(listener.games), listener.boards["board"])
	}
}

func TestListenerStaysSilentWhenPreferenceReadFails(t *testing.T) {
	listener, _ := testListener(t)
	listener.preferences.(*fakePreferences).err = errors.New("database unavailable")
	listener.handleMessageCreate(nil, commandEvent("$oc", "300"))
	listener.handleMessageCreate(nil, &discordgo.MessageCreate{Message: testBoardMessage("board", "", "grid-cell")})
	listener.mu.Lock()
	defer listener.mu.Unlock()
	if len(listener.games) != 0 || listener.boards["board"] != gameOC {
		t.Fatalf("games = %d, board = %d", len(listener.games), listener.boards["board"])
	}
}

func TestListenerPrunesExpiredCommandCorrelation(t *testing.T) {
	listener, _ := testListener(t)
	listener.recordCommand("300", gameOC, 1, 200, "command")
	listener.mu.Lock()
	listener.pending["300"][0].recorded = time.Now().Add(-commandCorrelationTTL - time.Second)
	listener.mu.Unlock()
	listener.handleMessageCreate(nil, &discordgo.MessageCreate{Message: testBoardMessage("late", "", "grid-cell")})
	listener.mu.Lock()
	defer listener.mu.Unlock()
	if len(listener.games) != 0 || len(listener.pending) != 0 {
		t.Fatalf("games = %d, pending = %d", len(listener.games), len(listener.pending))
	}
}

func TestListenerDoesNotCancelUnrelatedPendingCommand(t *testing.T) {
	listener, _ := testListener(t)
	listener.cancelOCCommand("300", "missing")
	listener.recordCommand("300", gameOH, 1, 200, "oh-command")
	listener.cancelOCCommand("300", "")
	listener.mu.Lock()
	defer listener.mu.Unlock()
	if len(listener.pending["300"]) != 1 || listener.pending["300"][0].kind != gameOH {
		t.Fatalf("pending = %#v", listener.pending["300"])
	}
}

func TestListenerSupportsCommandMultipliers(t *testing.T) {
	listener, messenger := testListener(t)
	listener.handleMessageCreate(nil, commandEvent("$oc 2", "300"))
	for _, id := range []string{"first", "second"} {
		listener.handleMessageCreate(nil, &discordgo.MessageCreate{Message: testBoardMessage(id, "", "grid-cell")})
	}
	receiveCall(t, messenger.sent)
	receiveCall(t, messenger.sent)
	listener.handleMessageDeleteBulk(nil, &discordgo.MessageDeleteBulk{Messages: []string{"first", "second"}})
	receiveCall(t, messenger.deleted)
	receiveCall(t, messenger.deleted)
}

func TestListenerExpiresAndDeletesHelper(t *testing.T) {
	listener, messenger := testListener(t)
	listener.gameTimeout = 15 * time.Millisecond
	listener.handleMessageCreate(nil, commandEvent("$oc", "300"))
	board := testBoardMessage("board", "Find the red sphere", "grid-cell")
	listener.handleMessageCreate(nil, &discordgo.MessageCreate{Message: board})
	receiveCall(t, messenger.sent)
	receiveCall(t, messenger.deleted)
}

func TestListenerDeletesHelperWhenGatewayContextStops(t *testing.T) {
	listener, messenger := testListener(t)
	ctx, cancel := context.WithCancel(context.Background())
	session, err := discordgo.New("Bot token")
	if err != nil {
		t.Fatal(err)
	}
	listener.Subscribe(ctx, session)
	listener.handleMessageCreate(nil, commandEvent("$oc", "300"))
	board := testBoardMessage("board", "Find the red sphere", "grid-cell")
	listener.handleMessageCreate(nil, &discordgo.MessageCreate{Message: board})
	receiveCall(t, messenger.sent)
	cancel()
	receiveCall(t, messenger.deleted)
}

func TestListenerContinuesAfterEditFailure(t *testing.T) {
	listener, messenger := testListener(t)
	listener.handleMessageCreate(nil, commandEvent("$oc", "300"))
	board := testBoardMessage("board", "Find the red sphere", "grid-cell")
	listener.handleMessageCreate(nil, &discordgo.MessageCreate{Message: board})
	sent := receiveCall(t, messenger.sent)
	messenger.mu.Lock()
	messenger.editErr = errors.New("unavailable")
	messenger.mu.Unlock()
	buttons, _ := flattenButtons(board.Components)
	buttons[0].Emoji = &discordgo.ComponentEmoji{ID: "101"}
	buttons[0].Disabled = true
	updated := *board
	updated.Components = buttonRows(buttons)
	listener.handleMessageUpdate(nil, &discordgo.MessageUpdate{Message: &updated})
	receiveCall(t, messenger.edited)
	listener.handleMessageDelete(nil, &discordgo.MessageDelete{Message: &discordgo.Message{ID: board.ID}})
	deleted := receiveCall(t, messenger.deleted)
	if deleted.messageID != sent.messageID {
		t.Fatalf("deleted = %#v", deleted)
	}
}

func TestListenerReportsInconsistentBoardInsteadOfGuessing(t *testing.T) {
	listener, messenger := testListener(t)
	listener.handleMessageCreate(nil, commandEvent("$oc", "300"))
	board := testBoardMessage("board", "Find the red sphere", "grid-cell")
	buttons, _ := flattenButtons(board.Components)
	for _, position := range []int{0, 1, 5} {
		buttons[position].Emoji = &discordgo.ComponentEmoji{ID: "105"}
		buttons[position].Disabled = true
	}
	board.Components = buttonRows(buttons)
	listener.handleMessageCreate(nil, &discordgo.MessageCreate{Message: board})
	call := receiveCall(t, messenger.sent)
	if !strings.Contains(call.content, "não vou arriscar") {
		t.Fatalf("content = %q", call.content)
	}
}

func TestListenerCleansStateAfterInitialSendFailure(t *testing.T) {
	listener, messenger := testListener(t)
	messenger.sendErr = errors.New("unavailable")
	listener.handleMessageCreate(nil, commandEvent("$oc", "300"))
	board := testBoardMessage("board", "Find the red sphere", "grid-cell")
	listener.handleMessageCreate(nil, &discordgo.MessageCreate{Message: board})
	receiveCall(t, messenger.sent)
	deadline := time.After(2 * time.Second)
	for {
		listener.mu.Lock()
		games := len(listener.games)
		listener.mu.Unlock()
		if games == 0 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("failed game was not removed")
		case <-time.After(time.Millisecond):
		}
	}
}

func TestNewValidatesIDsAndDependencies(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	emojis := testEmojiIDs()
	preferences, messages := &fakePreferences{automatic: true}, &fakeMessageLoader{}
	if _, err := New("invalid", emojis, &fakeMessenger{}, preferences, messages, logger); err == nil {
		t.Fatal("invalid Mudae ID accepted")
	}
	emojis.Red = emojis.Blue
	if _, err := New("100", emojis, &fakeMessenger{}, preferences, messages, logger); err == nil {
		t.Fatal("duplicate emoji accepted")
	}
	if _, err := New("100", testEmojiIDs(), nil, preferences, messages, logger); err == nil {
		t.Fatal("nil messenger accepted")
	}
	if _, err := New("100", testEmojiIDs(), &fakeMessenger{}, nil, messages, logger); err == nil {
		t.Fatal("nil preferences accepted")
	}
	if _, err := New("100", testEmojiIDs(), &fakeMessenger{}, preferences, nil, logger); err == nil {
		t.Fatal("nil message loader accepted")
	}
	if _, err := New("100", testEmojiIDs(), &fakeMessenger{}, preferences, messages, nil); err == nil {
		t.Fatal("nil logger accepted")
	}
}

func commandEvent(content, channelID string) *discordgo.MessageCreate {
	return &discordgo.MessageCreate{Message: &discordgo.Message{
		ID: "command-" + content, ChannelID: channelID, GuildID: "400", Content: content,
		Author: &discordgo.User{ID: "200"},
	}}
}

func testListener(t *testing.T) (*Listener, *fakeMessenger) {
	t.Helper()
	messenger := &fakeMessenger{
		sent: make(chan messageCall, 10), edited: make(chan messageCall, 10), deleted: make(chan messageCall, 10),
	}
	listener, err := New(
		"100", testEmojiIDs(), messenger, &fakePreferences{automatic: true},
		&fakeMessageLoader{}, slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	if err != nil {
		t.Fatal(err)
	}
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
	return listener, messenger
}

func testEmojiIDs() EmojiIDs {
	return EmojiIDs{Blue: "101", Teal: "102", Green: "103", Yellow: "104", Orange: "105", Red: "106"}
}

type messageCall struct {
	channelID string
	guildID   string
	sourceID  string
	messageID string
	content   string
}

type fakeMessenger struct {
	mu        sync.Mutex
	nextID    int
	sendErr   error
	editErr   error
	deleteErr error
	sent      chan messageCall
	edited    chan messageCall
	deleted   chan messageCall
}

type fakePreferences struct {
	mu        sync.Mutex
	automatic bool
	err       error
}

func (p *fakePreferences) AutoMudaeOC(uint64) (bool, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.automatic, p.err
}

func (p *fakePreferences) ToggleAutoMudaeOC(uint64) (bool, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.err != nil {
		return false, p.err
	}
	p.automatic = !p.automatic
	return p.automatic, nil
}

type fakeMessageLoader struct {
	messages map[string]*discordgo.Message
	err      error
}

func (l *fakeMessageLoader) LoadMessage(_, messageID string) (*discordgo.Message, error) {
	if l.err != nil {
		return nil, l.err
	}
	return l.messages[messageID], nil
}

func (m *fakeMessenger) SendReply(channelID, guildID, sourceMessageID, content string) (string, error) {
	m.mu.Lock()
	m.nextID++
	messageID := fmt.Sprintf("helper-%d", m.nextID)
	err := m.sendErr
	m.mu.Unlock()
	m.sent <- messageCall{channelID: channelID, guildID: guildID, sourceID: sourceMessageID, messageID: messageID, content: content}
	return messageID, err
}

func (m *fakeMessenger) EditMessage(channelID, messageID, content string) error {
	m.mu.Lock()
	err := m.editErr
	m.mu.Unlock()
	m.edited <- messageCall{channelID: channelID, messageID: messageID, content: content}
	return err
}

func (m *fakeMessenger) DeleteMessage(channelID, messageID string) error {
	m.mu.Lock()
	err := m.deleteErr
	m.mu.Unlock()
	m.deleted <- messageCall{channelID: channelID, messageID: messageID}
	return err
}

func receiveCall(t *testing.T, calls <-chan messageCall) messageCall {
	t.Helper()
	select {
	case call := <-calls:
		return call
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for Discord call")
		return messageCall{}
	}
}
