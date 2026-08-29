package discord

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/command"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/messagecommand"
)

func TestMessageCommandHandlerMapsAndSafelyReplies(t *testing.T) {
	requests := make(chan *http.Request, 1)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests <- request.Clone(request.Context())
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"id":"reply","channel_id":"channel","content":"pong"}`))
	}))
	defer server.Close()
	originalChannels := discordgo.EndpointChannels
	discordgo.EndpointChannels = server.URL + "/channels/"
	t.Cleanup(func() { discordgo.EndpointChannels = originalChannels })

	routes := NewRoutes()
	var received *messagecommand.Request
	routes.Messages().Command("!ping", func(_ context.Context, request *messagecommand.Request) error {
		received = request
		return request.Responder.Reply("pong")
	})
	if err := routes.Freeze(); err != nil {
		t.Fatal(err)
	}
	session, err := discordgo.New("Bot token")
	if err != nil {
		t.Fatal(err)
	}
	handler := &messageCommandHandler{
		routes: routes, logger: slog.New(slog.NewTextHandler(io.Discard, nil)), ctx: context.Background(),
	}
	handler.handle(session, &discordgo.MessageCreate{Message: &discordgo.Message{
		ID: "command", ChannelID: "channel", GuildID: "guild", Content: "!PING one",
		Author: &discordgo.User{ID: "123"}, Member: &discordgo.Member{Roles: []string{"role"}, Permissions: 8},
		MessageReference: &discordgo.MessageReference{MessageID: "source"},
	}})
	if received == nil || received.Actor.UserID != "123" || received.Actor.Permissions != 8 ||
		len(received.Actor.RoleIDs) != 1 || received.ReplyToID != "source" || len(received.Arguments) != 1 {
		t.Fatalf("request = %#v", received)
	}
	request := <-requests
	if request.Method != http.MethodPost || request.URL.Path != "/channels/channel/messages" {
		t.Fatalf("Discord request = %s %s", request.Method, request.URL.Path)
	}
}

func TestMessageCommandHandlerIgnoresIneligibleMessages(t *testing.T) {
	routes := NewRoutes()
	calls := 0
	routes.Messages().Command("!ping", func(context.Context, *messagecommand.Request) error { calls++; return nil })
	if err := routes.Freeze(); err != nil {
		t.Fatal(err)
	}
	handler := &messageCommandHandler{routes: routes, logger: slog.Default(), ctx: context.Background()}
	handler.handle(nil, nil)
	for _, message := range []*discordgo.Message{
		nil,
		{ID: "author", GuildID: "guild", Content: "!ping"},
		{ID: "dm", Content: "!ping", Author: &discordgo.User{ID: "1"}},
		{ID: "bot", GuildID: "guild", Content: "!ping", Author: &discordgo.User{ID: "1", Bot: true}},
		{ID: "webhook", GuildID: "guild", Content: "!ping", WebhookID: "hook", Author: &discordgo.User{ID: "1"}},
		{ID: "unknown", GuildID: "guild", Content: "!other", Author: &discordgo.User{ID: "1"}},
	} {
		handler.handle(nil, &discordgo.MessageCreate{Message: message})
	}
	if calls != 0 {
		t.Fatalf("calls = %d", calls)
	}
}

func TestMessageCommandHandlerReturnsExpectedAndGenericErrors(t *testing.T) {
	responses := make(chan string, 3)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		body, _ := io.ReadAll(request.Body)
		responses <- string(body)
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"id":"reply","channel_id":"channel"}`))
	}))
	defer server.Close()
	originalChannels := discordgo.EndpointChannels
	discordgo.EndpointChannels = server.URL + "/channels/"
	t.Cleanup(func() { discordgo.EndpointChannels = originalChannels })

	routes := NewRoutes()
	routes.Messages().Command("!reject", func(context.Context, *messagecommand.Request) error {
		return command.RejectForbidden("blocked")
	})
	routes.Messages().Command("!error", func(context.Context, *messagecommand.Request) error {
		return errors.New("boom")
	})
	routes.Messages().Command("!answered", func(_ context.Context, request *messagecommand.Request) error {
		if err := request.Responder.Reply("answer"); err != nil {
			return err
		}
		return errors.New("after response")
	})
	if err := routes.Freeze(); err != nil {
		t.Fatal(err)
	}
	session, err := discordgo.New("Bot token")
	if err != nil {
		t.Fatal(err)
	}
	handler := &messageCommandHandler{
		routes: routes, logger: slog.New(slog.NewTextHandler(io.Discard, nil)), ctx: context.Background(),
	}
	for _, trigger := range []string{"!reject", "!error", "!answered"} {
		handler.handle(session, &discordgo.MessageCreate{Message: &discordgo.Message{
			ID: trigger, ChannelID: "channel", GuildID: "guild", Content: trigger,
			Author: &discordgo.User{ID: "123"},
		}})
	}
	for _, want := range []string{"blocked", "Algo deu errado", "answer"} {
		if response := <-responses; !strings.Contains(response, want) {
			t.Fatalf("response = %q, want %q", response, want)
		}
	}
}

func TestMessageResponderAllowsOnlyOneReply(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"id":"reply","channel_id":"channel"}`))
	}))
	defer server.Close()
	originalChannels := discordgo.EndpointChannels
	discordgo.EndpointChannels = server.URL + "/channels/"
	t.Cleanup(func() { discordgo.EndpointChannels = originalChannels })
	session, err := discordgo.New("Bot token")
	if err != nil {
		t.Fatal(err)
	}
	responder := newMessageResponder(session, &discordgo.Message{ID: "command", ChannelID: "channel", GuildID: "guild"})
	if responder.responded() {
		t.Fatal("new responder already responded")
	}
	if err := responder.Reply("first"); err != nil || !responder.responded() {
		t.Fatalf("first Reply() error = %v, responded = %t", err, responder.responded())
	}
	if err := responder.Reply("second"); err == nil {
		t.Fatal("second Reply() error = nil")
	}
}

func TestMessageResponderAllowsOnlyUserMentionsWhenRequested(t *testing.T) {
	bodies := make(chan map[string]any, 1)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		bodies <- body
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"id":"reply","channel_id":"channel"}`))
	}))
	defer server.Close()
	originalChannels := discordgo.EndpointChannels
	discordgo.EndpointChannels = server.URL + "/channels/"
	t.Cleanup(func() { discordgo.EndpointChannels = originalChannels })
	session, err := discordgo.New("Bot token")
	if err != nil {
		t.Fatal(err)
	}
	responder := newMessageResponder(session, &discordgo.Message{ID: "command", ChannelID: "channel", GuildID: "guild"})
	if err := responder.ReplyWithUserMentions("> Test\n— <@123>"); err != nil {
		t.Fatal(err)
	}
	allowedMentions, ok := (<-bodies)["allowed_mentions"].(map[string]any)
	if !ok {
		t.Fatal("allowed_mentions is missing")
	}
	parse, ok := allowedMentions["parse"].([]any)
	if !ok || len(parse) != 1 || parse[0] != "users" || allowedMentions["replied_user"] != false {
		t.Fatalf("allowed_mentions = %#v", allowedMentions)
	}
}

func TestMessageCommandHandlerLogsFailedErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		http.Error(writer, `{"message":"bad request","code":0}`, http.StatusBadRequest)
	}))
	defer server.Close()
	originalChannels := discordgo.EndpointChannels
	discordgo.EndpointChannels = server.URL + "/channels/"
	t.Cleanup(func() { discordgo.EndpointChannels = originalChannels })
	routes := NewRoutes()
	routes.Messages().Command("!error", func(context.Context, *messagecommand.Request) error {
		return errors.New("boom")
	})
	if err := routes.Freeze(); err != nil {
		t.Fatal(err)
	}
	session, err := discordgo.New("Bot token")
	if err != nil {
		t.Fatal(err)
	}
	handler := &messageCommandHandler{
		routes: routes, logger: slog.New(slog.NewTextHandler(io.Discard, nil)), ctx: context.Background(),
	}
	handler.handle(session, &discordgo.MessageCreate{Message: &discordgo.Message{
		ID: "error", ChannelID: "channel", GuildID: "guild", Content: "!error",
		Author: &discordgo.User{ID: "123"},
	}})
}
