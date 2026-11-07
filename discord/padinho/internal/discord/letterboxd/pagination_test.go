package letterboxd

import (
	"context"
	"testing"

	"github.com/bwmarrin/discordgo"
	app "github.com/kiLeo13/SaladaFun/discord/padinho/internal/application/letterboxd"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/command"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/discord"
)

// TestReviewsPaginationUpdates separates the initial review reply from new and legacy page updates.
func TestReviewsPaginationUpdates(t *testing.T) {
	c := &catalog{}
	h := Handler{&app.Discovery{Catalog: c}}
	out := &visibilityResponder{}
	r := &discord.InteractionRequest{GuildID: "1", Actor: command.Actor{UserID: "11"}, Responder: out, Parameters: []string{"1", "1"}, Interaction: &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{}}}
	if err := h.reviews(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	if out.responseType != discordgo.InteractionResponseDeferredChannelMessageWithSource {
		t.Fatal("initial reviews should open a reply")
	}
	r.Interaction.Message = &discordgo.Message{Embeds: out.data.Embeds}
	if err := h.reviews(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	if out.responseType != discordgo.InteractionResponseDeferredMessageUpdate {
		t.Fatal("legacy pagination created a reply")
	}
	if err := h.reviewsPage(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	if out.responseType != discordgo.InteractionResponseDeferredMessageUpdate {
		t.Fatal("pagination created a reply")
	}
	for _, c := range out.data.Components[0].(discordgo.ActionsRow).Components {
		if id := c.(discordgo.Button).CustomID; id != "lb.reviews-page:1:0" && id != "lb.reviews-page:1:2" {
			t.Fatal("wrong page route", id)
		}
	}
}
