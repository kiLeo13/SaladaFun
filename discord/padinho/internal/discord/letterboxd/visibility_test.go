package letterboxd

import (
	"context"
	"testing"

	"github.com/bwmarrin/discordgo"
	app "github.com/kiLeo13/SaladaFun/discord/padinho/internal/application/letterboxd"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/command"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/discord"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
)

type visibilityResponder struct {
	responder
	flags        discordgo.MessageFlags
	responseType discordgo.InteractionResponseType
}

// Respond retains acknowledgement flags before Complete replaces the captured payload.
func (r *visibilityResponder) Respond(response *discordgo.InteractionResponse) error {
	r.flags = 0
	r.responseType = response.Type
	if response.Data != nil {
		r.flags = response.Data.Flags
	}
	return r.responder.Respond(response)
}

// TestComponentVisibility checks public browsing and private personal actions using local fakes.
func TestComponentVisibility(t *testing.T) {
	for _, tc := range []struct {
		name       string
		parameters []string
		private    bool
	}{
		{"select", []string{"show"}, false},
		{"select", []string{"add"}, true},
		{"select", []string{"remove"}, true},
		{"page", []string{"show", "2"}, false},
		{"page", []string{"list", "1"}, false},
		{"page", []string{"remove", "1"}, false},
		{"reviews", []string{"1", "1"}, false},
		{"reviews", []string{"1", "2"}, false},
	} {
		t.Run(tc.name+"/"+tc.parameters[0], func(t *testing.T) {
			c := &catalog{items: []entity.LetterboxdTitle{{ID: 1, TMDBID: 1, MediaType: "movie", OriginalTitle: "Title"}}}
			reviews := reviewCatalog{catalog: c, opinions: []app.Opinion{{UserID: 1, Rating: 5}, {UserID: 2, Rating: 4}, {UserID: 3, Rating: 3}, {UserID: 4, Rating: 2}}}
			h := Handler{&app.Discovery{Catalog: reviews, Metadata: app.New(c, c, nil), SearchProvider: c}}
			out := &visibilityResponder{}
			r := &discord.InteractionRequest{GuildID: "1", Actor: command.Actor{UserID: "11"}, Responder: out, Parameters: tc.parameters, Interaction: &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{Type: discordgo.InteractionMessageComponent, Data: discordgo.MessageComponentInteractionData{Values: []string{"movie:1"}}}}}
			var err error
			switch tc.name {
			case "select":
				err = h.selectTitle(context.Background(), r)
			case "page":
				err = h.changePage(context.Background(), r)
			case "reviews":
				err = h.reviews(context.Background(), r)
			}
			if err != nil {
				t.Fatal(err)
			}
			if private := out.flags&discordgo.MessageFlagsEphemeral != 0; private != tc.private || out.initial != 1 {
				t.Fatalf("acknowledgement: private=%v, initial=%d", private, out.initial)
			}
			if tc.name == "page" && out.responseType != discordgo.InteractionResponseDeferredMessageUpdate {
				t.Fatal("pagination created a new reply")
			}
		})
	}
}
