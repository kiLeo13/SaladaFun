package discord

import (
	"context"
	"github.com/bwmarrin/discordgo"
	"testing"
)

// TestAutocompleteDispatch verifies partial grouped options bypass full validation.
func TestAutocompleteDispatch(t *testing.T) {
	routes := NewRoutes()
	var called bool
	routes.Autocomplete("letterboxd.wishlist.add.title", func(_ context.Context, r *AutocompleteRequest) error {
		called = true
		q, _ := r.Options.String("title")
		if q != "Boot" || r.Focused != "title" || r.Path.Group != "wishlist" {
			t.Fatal("mapping")
		}
		return nil
	})
	routes.Freeze()
	interaction := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{Type: discordgo.InteractionApplicationCommandAutocomplete, Data: discordgo.ApplicationCommandInteractionData{Name: "letterboxd", Options: []*discordgo.ApplicationCommandInteractionDataOption{{Name: "wishlist", Type: discordgo.ApplicationCommandOptionSubCommandGroup, Options: []*discordgo.ApplicationCommandInteractionDataOption{{Name: "add", Type: discordgo.ApplicationCommandOptionSubCommand, Options: []*discordgo.ApplicationCommandInteractionDataOption{{Name: "title", Type: discordgo.ApplicationCommandOptionString, Value: "Boot", Focused: true}}}}}}}}}
	handled, err := routes.dispatch(context.Background(), interaction, &testResponder{})
	if !handled || err != nil || !called {
		t.Fatalf("dispatch %v", err)
	}
	interaction.Data = discordgo.ApplicationCommandInteractionData{Name: "unknown"}
	if _, err := routes.dispatch(context.Background(), interaction, &testResponder{}); err != ErrUnknownRoute {
		t.Fatal("unknown")
	}
	interaction.Data = discordgo.ApplicationCommandInteractionData{Name: "x", Options: []*discordgo.ApplicationCommandInteractionDataOption{{Name: "group", Type: discordgo.ApplicationCommandOptionSubCommandGroup}}}
	if _, err := routes.dispatch(context.Background(), interaction, &testResponder{}); err == nil {
		t.Fatal("malformed")
	}

}

// TestAutocompleteRegistration rejects invalid or mutable declarations.
func TestAutocompleteRegistration(t *testing.T) {
	for _, scenario := range []string{"empty", "nil", "duplicate", "frozen"} {
		t.Run(scenario, func(t *testing.T) {
			r := NewRoutes()
			h := func(context.Context, *AutocompleteRequest) error { return nil }
			defer func() {
				if recover() == nil {
					t.Fatal("expected rejection")
				}
			}()
			switch scenario {
			case "empty":
				r.Autocomplete("", h)
			case "nil":
				r.Autocomplete("x", nil)
			case "duplicate":
				r.Autocomplete("x", h)
				r.Autocomplete("x", h)
			case "frozen":
				r.Freeze()
				r.Autocomplete("x", h)
			}
		})
	}
}
