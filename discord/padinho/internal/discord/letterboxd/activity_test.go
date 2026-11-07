package letterboxd

import (
	"context"
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
	app "github.com/kiLeo13/SaladaFun/discord/padinho/internal/application/letterboxd"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/command"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/discord"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/locale/ptbr"
)

type activityStore struct {
	user        uint64
	writes      int
	rating      int
	text        string
	empty       bool
	sort, order string
	page        int
}

// Watch supplies only the actor's opinion for modal prefilling.
func (s *activityStore) Watch(_ context.Context, user, title uint64) (*app.WatchState, error) {
	text := "Old review"
	state := &app.WatchState{Title: entity.LetterboxdTitle{ID: title}}
	if user == 11 {
		state.Rating = 4
		state.Review = &text
	}
	return state, nil
}

// SaveWatch captures submission ownership and optional text.
func (s *activityStore) SaveWatch(_ context.Context, user, title, interaction uint64, rating int, text string) error {
	s.user = user
	s.writes++
	s.rating = rating
	s.text = text
	return nil
}

// History captures ordering and includes a sentinel lookahead row.
func (s *activityStore) History(_ context.Context, user uint64, sort, order string, page int) ([]app.HistoryEntry, error) {
	s.user = user
	s.sort = sort
	s.order = order
	s.page = page
	if s.empty {
		return nil, nil
	}
	text := strings.Repeat("*", 500)
	return []app.HistoryEntry{{ID: 1, OriginalTitle: "Title", Review: &text}, {ID: 2}, {ID: 3}, {ID: 4}}, nil
}

// RandomUnwatched supplies a candidate independently of remote search.
func (s *activityStore) RandomUnwatched(_ context.Context, user uint64) (*entity.LetterboxdTitle, error) {
	s.user = user
	if s.empty {
		return nil, nil
	}
	return &entity.LetterboxdTitle{ID: 1, TMDBID: 1, MediaType: "movie"}, nil
}

// TestWatchModalFlow covers prefilling, cancellation, actor ownership and withdrawal.
func TestWatchModalFlow(t *testing.T) {
	ctx := context.Background()
	c := &catalog{}
	store := &activityStore{}
	h := Handler{&app.Discovery{Metadata: app.New(c, c, nil), Catalog: c, Activity: &app.Activity{Store: store}}}
	out := &responder{}
	request := &discord.InteractionRequest{GuildID: "1", Actor: command.Actor{UserID: "11"}, Parameters: []string{"1"}, Responder: out}
	if err := h.openWatch(ctx, request); err != nil || store.writes != 0 {
		t.Fatal("open wrote state", err)
	}
	fields := out.data.Components
	selectMenu := fields[0].(discordgo.Label).Component.(discordgo.SelectMenu)
	if !selectMenu.Options[3].Default || len(selectMenu.Options) != 5 {
		t.Fatal("rating prefill")
	}
	if fields[1].(discordgo.Label).Component.(discordgo.TextInput).Value != "Old review" {
		t.Fatal("review prefill")
	}
	request.Actor.UserID = "12"
	if err := h.openWatch(ctx, request); err != nil {
		t.Fatal(err)
	}
	if out.data.Components[1].(discordgo.Label).Component.(discordgo.TextInput).Value != "" {
		t.Fatal("leaked another member review")
	}
	request.Interaction = &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{Type: discordgo.InteractionModalSubmit, ID: "100", Data: discordgo.ModalSubmitInteractionData{Components: []discordgo.MessageComponent{discordgo.Label{Component: discordgo.SelectMenu{CustomID: "rating", Values: []string{"5"}}}, discordgo.Label{Component: discordgo.TextInput{CustomID: "review", Value: ""}}}}}}
	if err := h.saveWatch(ctx, request); err != nil || store.user != 12 || store.writes != 1 || store.rating != 5 || store.text != "" {
		t.Fatal("submission", err)
	}
	request.Parameters = []string{"0"}
	if err := h.openWatch(ctx, request); err == nil {
		t.Fatal("forged title")
	}
	request.Parameters = []string{"1"}
	request.GuildID = ""
	if err := h.saveWatch(ctx, request); err == nil {
		t.Fatal("DM submission")
	}
}

// TestModalValidation rejects forged ratings, duplicate fields and overlong reviews.
func TestModalValidation(t *testing.T) {
	for _, fields := range [][]discordgo.MessageComponent{
		nil, {discordgo.SelectMenu{CustomID: "rating", Values: []string{"0"}}}, {discordgo.SelectMenu{CustomID: "rating", Values: []string{"6"}}}, {discordgo.SelectMenu{CustomID: "rating", Values: []string{"x"}}}, {discordgo.SelectMenu{CustomID: "rating", Values: []string{"1", "2"}}}, {discordgo.SelectMenu{CustomID: "rating", Values: []string{"1"}}, discordgo.SelectMenu{CustomID: "rating", Values: []string{"2"}}}, {discordgo.SelectMenu{CustomID: "rating", Values: []string{"1"}}, discordgo.TextInput{CustomID: "review", Value: "x"}}, {discordgo.SelectMenu{CustomID: "rating", Values: []string{"1"}}, discordgo.TextInput{CustomID: "review", Value: strings.Repeat("x", 501)}}, {discordgo.TextInput{CustomID: "forged"}},
	} {
		if _, _, err := modalFields(fields); err == nil {
			t.Fatal("malformed modal accepted")
		}
	}
	fields := []discordgo.MessageComponent{&discordgo.Label{Component: &discordgo.ActionsRow{Components: []discordgo.MessageComponent{&discordgo.SelectMenu{CustomID: "rating", Values: []string{"3"}}, &discordgo.TextInput{CustomID: "review", Value: "Good"}}}}}
	if rating, text, err := modalFields(fields); err != nil || rating != 3 || text != "Good" {
		t.Fatal("native pointers")
	}
}

// TestHistoryAndRandom verifies defaults, explicit members, safe text and empty pools.
func TestHistoryAndRandom(t *testing.T) {
	c := &catalog{}
	store := &activityStore{}
	h := Handler{&app.Discovery{Metadata: app.New(c, c, nil), Catalog: c, Activity: &app.Activity{Store: store}}}
	out := &responder{}
	ctx := context.Background()
	request := &command.CommandRequest{GuildID: "1", Actor: command.Actor{UserID: "11"}, Responder: out, Options: command.NewOptionValues(nil)}
	if err := h.history(ctx, request); err != nil || store.user != 11 || store.sort != "date" || store.order != "desc" || len(out.data.Embeds) != 3 {
		t.Fatal("defaults", err)
	}
	if !strings.Contains(out.data.Embeds[0].Description, "\\*") || len(out.data.Embeds[0].Description) > 4096 {
		t.Fatal("review render")
	}
	request.Options = command.NewOptionValues(map[string]any{"member": command.Snowflake("12"), "sort": "rating", "order": "asc"})
	if err := h.history(ctx, request); err != nil || store.user != 12 || store.sort != "rating" || store.order != "asc" {
		t.Fatal("options")
	}
	nav := &discord.InteractionRequest{GuildID: "1", Actor: command.Actor{UserID: "11"}, Parameters: []string{"12", "rating", "asc", "2"}, Responder: out}
	if err := h.historyPage(ctx, nav); err != nil || store.page != 2 {
		t.Fatal("navigation")
	}
	nav.Parameters[1] = "DROP TABLE"
	if err := h.historyPage(ctx, nav); err == nil {
		t.Fatal("forged sort")
	}
	if err := h.pickRandom(ctx, request); err != nil || store.user != 11 {
		t.Fatal("random actor", err)
	}
	store.empty = true
	if err := h.pickRandom(ctx, request); err != nil || out.data.Content != ptbr.LetterboxdNoRandom {
		t.Fatal("empty random")
	}
	if err := h.history(ctx, request); err != nil || !strings.Contains(out.data.Content, ptbr.LetterboxdNoHistory) {
		t.Fatal("empty history")
	}
}
