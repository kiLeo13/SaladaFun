package letterboxd

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf16"

	"github.com/bwmarrin/discordgo"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/command"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/locale/ptbr"
)

// checkAboutPayload validates the Components V2-only layout and its serialized image accessory.
func checkAboutPayload(t *testing.T, data *discordgo.InteractionResponseData) {
	t.Helper()
	if data.Flags&discordgo.MessageFlagsIsComponentsV2 == 0 || data.Content != "" || len(data.Embeds) != 0 || len(data.Components) != 1 {
		t.Fatal("invalid V2 payload")
	}
	if data.AllowedMentions == nil {
		t.Fatal("missing mention suppression")
	}
	container, ok := data.Components[0].(discordgo.Container)
	if !ok || len(container.Components) != 4 {
		t.Fatal("about container")
	}
	for i, want := range []string{ptbr.LetterboxdAboutOverview, ptbr.LetterboxdAboutLocal} {
		block, ok := container.Components[i].(discordgo.TextDisplay)
		if !ok || block.Content != want || len(utf16.Encode([]rune(block.Content))) > 4000 {
			t.Fatal("about text block", i)
		}
	}
	if separator, ok := container.Components[2].(discordgo.Separator); !ok || separator.Divider == nil || !*separator.Divider {
		t.Fatal("attribution separator")
	}
	section, ok := container.Components[3].(discordgo.Section)
	if !ok || len(section.Components) != 1 {
		t.Fatal("attribution section")
	}
	text := section.Components[0].(discordgo.TextDisplay).Content
	if !strings.Contains(text, ptbr.LetterboxdCredit) || !strings.Contains(text, tmdbWebsiteURL) {
		t.Fatal("missing provider attribution")
	}
	logo, ok := section.Accessory.(discordgo.Thumbnail)
	if !ok || logo.Media.URL != tmdbLogoURL || logo.Description == nil || *logo.Description != ptbr.LetterboxdTMDBLogo {
		t.Fatal("provider logo")
	}
	var raw map[string]json.RawMessage
	payload, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		t.Fatal(err)
	}
	if embeds, exists := raw["embeds"]; exists && string(embeds) != "null" && string(embeds) != "[]" {
		t.Fatal("nonempty legacy embeds serialized")
	}
	if content, exists := raw["content"]; exists && string(content) != `""` {
		t.Fatal("nonempty legacy content serialized")
	}
}

// TestAboutComponentsV2 verifies the localized informational page without invoking services.
func TestAboutComponentsV2(t *testing.T) {
	out := &responder{}
	r := &command.CommandRequest{GuildID: "1", Responder: out}
	h := Handler{}
	if err := h.about(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	if out.initial != 1 || out.data.Flags&discordgo.MessageFlagsEphemeral == 0 {
		t.Fatal("about visibility changed")
	}
	checkAboutPayload(t, out.data)
	r.GuildID = ""
	if err := h.about(context.Background(), r); err == nil || out.initial != 1 {
		t.Fatal("about accepted a DM")
	}
}
