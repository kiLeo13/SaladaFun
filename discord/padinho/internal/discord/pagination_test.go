package discord

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
)

// TestDeferredSourceUpdate edits @original after one type-6 callback without creating a message.
func TestDeferredSourceUpdate(t *testing.T) {
	session, _ := discordgo.New("Bot token")
	callbacks, edits := 0, 0
	session.Client = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/callback") {
			callbacks++
			var body struct{ Type int }
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Type != 6 {
				t.Fatal("update callback", body, err)
			}
			return &http.Response{StatusCode: 204, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header), Request: r}, nil
		}
		if r.Method != http.MethodPatch || !strings.HasSuffix(r.URL.Path, "/messages/@original") {
			t.Fatal("unexpected message creation", r.Method, r.URL.Path)
		}
		edits++
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"id":"source"}`)), Header: make(http.Header), Request: r}, nil
	})}
	r := newInteractionResponder(session, &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{ID: "1", AppID: "app", Token: "token"}})
	if err := r.Respond(&discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate}); err != nil {
		t.Fatal(err)
	}
	if err := r.Complete(&discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{{Title: "Next page"}}}); err != nil {
		t.Fatal(err)
	}
	if callbacks != 1 || edits != 1 || r.deferred {
		t.Fatal("update lifecycle", callbacks, edits)
	}
}

// TestPaginationErrorPreservesSource retains the current page and controls when rendering fails.
func TestPaginationErrorPreservesSource(t *testing.T) {
	session, _ := discordgo.New("Bot token")
	edited := false
	session.Client = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		body := ""
		status := 204
		if r.Method == http.MethodPatch {
			edited = true
			var data struct {
				Embeds     []*discordgo.MessageEmbed
				Components []json.RawMessage
			}
			if err := json.NewDecoder(r.Body).Decode(&data); err != nil || len(data.Embeds) != 1 || data.Embeds[0].Title != "Current page" || len(data.Components) != 1 {
				t.Fatal("lost source on error", data, err)
			}
			body = `{"id":"source"}`
			status = 200
		}
		return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header), Request: r}, nil
	})}
	routes := NewRoutes()
	routes.Component("page", func(_ context.Context, r *InteractionRequest) error {
		if err := r.Responder.Respond(&discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate}); err != nil {
			return err
		}
		return errors.New("page failed")
	})
	if err := routes.Freeze(); err != nil {
		t.Fatal(err)
	}
	h := &interactionHandler{routes: routes, ctx: context.Background(), logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	h.handle(session, &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{ID: "1", AppID: "app", Token: "token", Type: discordgo.InteractionMessageComponent, Data: discordgo.MessageComponentInteractionData{CustomID: "page:next"}, Message: &discordgo.Message{Embeds: []*discordgo.MessageEmbed{{Title: "Current page"}}, Components: []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.Button{CustomID: "page:next", Style: discordgo.SecondaryButton, Label: "Next"}}}}}}})
	if !edited {
		t.Fatal("error did not update the current message")
	}
}
