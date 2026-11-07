package letterboxd

import (
	"context"
	"github.com/bwmarrin/discordgo"
	app "github.com/kiLeo13/SaladaFun/discord/padinho/internal/application/letterboxd"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/command"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/discord"
	"testing"
)

// TestLegacyReviewPrefill flattens old multiline text in the edit modal without saving.
func TestLegacyReviewPrefill(t *testing.T) {
	text := " \r Old\n\n review \r\n "
	store := &activityStore{reviewText: &text}
	h := Handler{&app.Discovery{Activity: &app.Activity{Store: store}}}
	out := &responder{}
	r := &discord.InteractionRequest{GuildID: "1", Actor: command.Actor{UserID: "11"}, Responder: out, Parameters: []string{"1"}}
	if err := h.openWatch(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	checkPayload(t, out.data)
	field := out.data.Components[1].(discordgo.Label).Component.(discordgo.TextInput)
	if field.Value != "Old review" || store.writes != 0 || text != " \r Old\n\n review \r\n " {
		t.Fatal("legacy modal prefill", field.Value)
	}
}

// TestRenderedOpinionsNormalizeLegacyWhitespace covers shared reviews and personal history text.
func TestRenderedOpinionsNormalizeLegacyWhitespace(t *testing.T) {
	if got := plainOpinion(" \r\n A\rB\n\n C >x \r "); got != "A B C \\>x" {
		t.Fatal(got)
	}
}
