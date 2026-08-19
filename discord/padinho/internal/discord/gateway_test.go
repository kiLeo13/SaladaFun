package discord

import (
	"io"
	"log/slog"
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestNewRequestsGuildAndVoiceStateIntents(t *testing.T) {
	gateway, err := New("token", NewRoutes(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	if gateway.session.Identify.Intents&(discordgo.IntentsGuilds|discordgo.IntentsGuildVoiceStates) != discordgo.IntentsGuilds|discordgo.IntentsGuildVoiceStates {
		t.Fatalf("gateway intents = %v", gateway.session.Identify.Intents)
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
