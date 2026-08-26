package discord

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestNewRequestsRequiredGatewayIntents(t *testing.T) {
	gateway, err := New("token", NewRoutes(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	want := discordgo.IntentsGuilds | discordgo.IntentsGuildVoiceStates |
		discordgo.IntentsGuildMessages | discordgo.IntentsMessageContent
	if gateway.session.Identify.Intents&want != want {
		t.Fatalf("gateway intents = %v", gateway.session.Identify.Intents)
	}
}

func TestAddSubscriber(t *testing.T) {
	gateway, err := New("token", NewRoutes(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	subscriber := stubSubscriber{}
	if err := gateway.AddSubscriber(subscriber); err != nil {
		t.Fatal(err)
	}
	if len(gateway.subscribers) != 1 {
		t.Fatalf("subscribers = %d", len(gateway.subscribers))
	}
	if err := gateway.AddSubscriber(nil); err == nil {
		t.Fatal("AddSubscriber(nil) error = nil")
	}
}

func TestGuildReadsCachedGuildAndHandlesMissingGuild(t *testing.T) {
	gateway, err := New("token", NewRoutes(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	want := &discordgo.Guild{ID: "guild", Name: "Salada"}
	if err := gateway.session.State.GuildAdd(want); err != nil {
		t.Fatal(err)
	}
	got, err := gateway.Guild("guild")
	if err != nil || got != want {
		t.Fatalf("Guild() = %#v, %v", got, err)
	}
	missing, err := gateway.Guild("missing")
	if err != nil || missing != nil {
		t.Fatalf("missing Guild() = %#v, %v", missing, err)
	}
}

func TestApplicationID(t *testing.T) {
	session := &discordgo.Session{State: discordgo.NewState()}
	if got := applicationID(session); got != "" {
		t.Fatalf("applicationID() = %q", got)
	}
	session.State.User = &discordgo.User{ID: "application"}
	if got := applicationID(session); got != "application" {
		t.Fatalf("applicationID() = %q", got)
	}
}

func TestReplyLifecycleUsesSafeDiscordPayloads(t *testing.T) {
	type requestRecord struct {
		method string
		path   string
		body   []byte
	}
	requests := make(chan requestRecord, 3)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		body, _ := io.ReadAll(request.Body)
		requests <- requestRecord{method: request.Method, path: request.URL.Path, body: body}
		writer.Header().Set("Content-Type", "application/json")
		if request.Method == http.MethodDelete {
			writer.WriteHeader(http.StatusNoContent)
			return
		}
		_, _ = writer.Write([]byte(`{"id":"helper","channel_id":"channel","content":"suggestion"}`))
	}))
	defer server.Close()

	originalChannels := discordgo.EndpointChannels
	discordgo.EndpointChannels = server.URL + "/channels/"
	t.Cleanup(func() { discordgo.EndpointChannels = originalChannels })

	gateway, err := New("token", NewRoutes(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	helperID, err := gateway.SendReply("channel", "guild", "source", "suggestion")
	if err != nil || helperID != "helper" {
		t.Fatalf("SendReply() = %q, %v", helperID, err)
	}
	if err := gateway.EditMessage("channel", helperID, "updated"); err != nil {
		t.Fatal(err)
	}
	if err := gateway.DeleteMessage("channel", helperID); err != nil {
		t.Fatal(err)
	}

	create := <-requests
	if create.method != http.MethodPost || create.path != "/channels/channel/messages" {
		t.Fatalf("create request = %s %s", create.method, create.path)
	}
	var body struct {
		AllowedMentions discordgo.MessageAllowedMentions `json:"allowed_mentions"`
		Reference       discordgo.MessageReference       `json:"message_reference"`
	}
	if err := json.Unmarshal(create.body, &body); err != nil {
		t.Fatal(err)
	}
	if len(body.AllowedMentions.Parse) != 0 || body.Reference.MessageID != "source" ||
		body.Reference.ChannelID != "channel" || body.Reference.GuildID != "guild" ||
		body.Reference.FailIfNotExists == nil || *body.Reference.FailIfNotExists {
		t.Fatalf("unsafe reply payload = %#v", body)
	}
	edit := <-requests
	if edit.method != http.MethodPatch || edit.path != "/channels/channel/messages/helper" {
		t.Fatalf("edit request = %s %s", edit.method, edit.path)
	}
	remove := <-requests
	if remove.method != http.MethodDelete || remove.path != "/channels/channel/messages/helper" {
		t.Fatalf("delete request = %s %s", remove.method, remove.path)
	}
}

func TestReplyLifecycleWrapsDiscordErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		http.Error(writer, `{"message":"bad request","code":0}`, http.StatusBadRequest)
	}))
	defer server.Close()
	originalChannels := discordgo.EndpointChannels
	discordgo.EndpointChannels = server.URL + "/channels/"
	t.Cleanup(func() { discordgo.EndpointChannels = originalChannels })

	gateway, err := New("token", NewRoutes(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := gateway.SendReply("channel", "guild", "source", "suggestion"); err == nil ||
		!strings.Contains(err.Error(), "send Discord reply") {
		t.Fatalf("SendReply() error = %v", err)
	}
	if err := gateway.EditMessage("channel", "helper", "updated"); err == nil ||
		!strings.Contains(err.Error(), "edit Discord message") {
		t.Fatalf("EditMessage() error = %v", err)
	}
	if err := gateway.DeleteMessage("channel", "helper"); err == nil ||
		!strings.Contains(err.Error(), "delete Discord message") {
		t.Fatalf("DeleteMessage() error = %v", err)
	}
	if _, err := gateway.LoadMessage("channel", "source"); err == nil ||
		!strings.Contains(err.Error(), "load Discord message") {
		t.Fatalf("LoadMessage() error = %v", err)
	}
}

func TestLoadMessageReturnsCurrentDiscordPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet || request.URL.Path != "/channels/channel/messages/source" {
			t.Fatalf("request = %s %s", request.Method, request.URL.Path)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"id":"source","channel_id":"channel","guild_id":"guild","content":"$oc"}`))
	}))
	defer server.Close()
	originalChannels := discordgo.EndpointChannels
	discordgo.EndpointChannels = server.URL + "/channels/"
	t.Cleanup(func() { discordgo.EndpointChannels = originalChannels })
	gateway, err := New("token", NewRoutes(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	message, err := gateway.LoadMessage("channel", "source")
	if err != nil || message.ID != "source" || message.Content != "$oc" {
		t.Fatalf("LoadMessage() = %#v, %v", message, err)
	}
}

type stubSubscriber struct{}

func (stubSubscriber) Subscribe(context.Context, *discordgo.Session) {}
