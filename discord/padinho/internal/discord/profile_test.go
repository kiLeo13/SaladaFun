package discord

import (
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/command"
)

// TestInteractionProfiles retains caller and selected-member data without a user lookup.
func TestInteractionProfiles(t *testing.T) {
	caller := &discordgo.User{ID: "11", Username: "caller", GlobalName: "Caller", Avatar: "abc"}
	selected := &discordgo.User{ID: "22", Username: "selected", GlobalName: "Selected", Avatar: "def"}
	interaction := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		Type: discordgo.InteractionApplicationCommand, Member: &discordgo.Member{User: caller},
		Data: discordgo.ApplicationCommandInteractionData{Name: "letterboxd", Resolved: &discordgo.ApplicationCommandInteractionDataResolved{Users: map[string]*discordgo.User{"22": selected, "33": nil}}},
	}}
	r, err := mapRequest(interaction, &testResponder{})
	if err != nil {
		t.Fatal(err)
	}
	for id, user := range map[string]*discordgo.User{"11": caller, "22": selected} {
		profile := r.Users[command.Snowflake(id)]
		if profile.GlobalName != user.GlobalName || profile.Username != user.Username || profile.AvatarURL != user.AvatarURL("128") {
			t.Fatalf("profile %s: %#v", id, profile)
		}
	}
	if _, exists := r.Users["33"]; exists {
		t.Fatal("nil resolved user included")
	}
	interaction.Member = nil
	interaction.User = caller
	interaction.Data = discordgo.ApplicationCommandInteractionData{Name: "letterboxd"}
	if profiles := interactionProfiles(interaction); profiles["11"].GlobalName != "Caller" {
		t.Fatal("direct caller")
	}
	interaction.User = nil
	if profiles := interactionProfiles(interaction); len(profiles) != 0 {
		t.Fatal("invented profile")
	}
}
