package discord

import (
	"testing"

	"github.com/bwmarrin/discordgo"
)

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
