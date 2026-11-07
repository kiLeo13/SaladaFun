package letterboxd

import (
	"context"
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
	app "github.com/kiLeo13/SaladaFun/discord/padinho/internal/application/letterboxd"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/command"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/discord"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/locale/ptbr"
)

// TestHistoryAuthorFallbacks always produces an author with the selected member's identity.
func TestHistoryAuthorFallbacks(t *testing.T) {
	for _, tc := range []struct {
		profile command.UserProfile
		want    string
	}{
		{command.UserProfile{GlobalName: "Global", Username: "user", AvatarURL: "https://example.test/avatar.png"}, "Avaliações de Global"},
		{command.UserProfile{Username: "user"}, "Avaliações de user"},
		{command.UserProfile{}, "Avaliações de 11"},
	} {
		author := historyAuthor(11, map[command.Snowflake]command.UserProfile{"11": tc.profile})
		if author.Name != tc.want || author.IconURL != tc.profile.AvatarURL || author.URL != "https://discord.com/users/11" {
			t.Fatalf("author %#v", author)
		}
	}
}

// TestPublicHistoryLayout verifies one embed, totals, author reuse and maximum text budgets.
func TestPublicHistoryLayout(t *testing.T) {
	store := &activityStore{}
	h := Handler{&app.Discovery{Activity: &app.Activity{Store: store}}}
	out := &visibilityResponder{}
	r := &command.CommandRequest{GuildID: "1", Actor: command.Actor{UserID: "11"}, Responder: out, Options: command.NewOptionValues(nil), Users: map[command.Snowflake]command.UserProfile{"11": {GlobalName: "Leo", AvatarURL: "https://example.test/avatar.png"}}}
	if err := h.history(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	if out.flags&discordgo.MessageFlagsEphemeral != 0 || len(out.data.Embeds) != 1 || out.data.Content != "" {
		t.Fatal("history should be one public embed")
	}
	checkPayload(t, out.data)
	embed := out.data.Embeds[0]
	if embed.Author.Name != "Avaliações de Leo" || embed.Footer.Text != "Total: 4 avaliações." || !strings.Contains(embed.Description, "**Title**: **5/5** · <t:1:d>\n> ") {
		t.Fatalf("history %#v", embed)
	}
	if strings.Contains(embed.Description, "(2020") || !strings.Contains(embed.Description, "\n\n") {
		t.Fatal("year or spacing")
	}
	nav := &discord.InteractionRequest{GuildID: "1", Actor: command.Actor{UserID: "99"}, Responder: out, Parameters: []string{"11", "date", "desc", "2"}, Interaction: &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{Message: &discordgo.Message{Embeds: out.data.Embeds}}}}
	if err := h.historyPage(context.Background(), nav); err != nil {
		t.Fatal(err)
	}
	if out.flags&discordgo.MessageFlagsEphemeral != 0 || out.data.Embeds[0].Author.Name != "Avaliações de Leo" || out.data.Embeds[0].Footer.Text != "Total: 4 avaliações." {
		t.Fatal("pagination lost member or total")
	}
	nav.Interaction.Message.Embeds = nil
	if err := h.historyPage(context.Background(), nav); err != nil {
		t.Fatal(err)
	}
	if out.data.Embeds[0].Author.Name != "Avaliações de 11" {
		t.Fatal("snowflake fallback")
	}
	store.empty = true
	if err := h.history(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	if len(out.data.Embeds) != 1 || out.data.Embeds[0].Description != ptbr.LetterboxdNoHistory || out.data.Embeds[0].Footer.Text != "Total: 0 avaliações." {
		t.Fatal("empty history")
	}
	h.service.Activity.Store = boundaryHistoryStore{store}
	if err := h.renderHistory(context.Background(), out, 11, "date", "desc", 1); err != nil {
		t.Fatal(err)
	}
	checkPayload(t, out.data)
	if len(out.data.Embeds) != 1 {
		t.Fatal("worst case split into multiple embeds")
	}
}
