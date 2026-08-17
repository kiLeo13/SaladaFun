package discord

import (
	"io"
	"log/slog"
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestNewRequestsVoiceStateIntent(t *testing.T) {
	gateway, err := New("token", NewRoutes(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	if gateway.session.Identify.Intents&discordgo.IntentsGuildVoiceStates == 0 {
		t.Fatal("gateway does not request voice-state events")
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
