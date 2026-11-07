package discord

import (
	"context"
	"errors"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/command"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/locale/ptbr"
)

func TestInteractionResponderSendsOneNativeResponse(t *testing.T) {
	var body string
	session, err := discordgo.New("Bot token")
	if err != nil {
		t.Fatal(err)
	}
	session.Client = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		payload, readErr := io.ReadAll(request.Body)
		if readErr != nil {
			return nil, readErr
		}
		body = string(payload)
		return &http.Response{
			StatusCode: http.StatusNoContent,
			Body:       io.NopCloser(strings.NewReader("")), Header: make(http.Header),
			Request: request,
		}, nil
	})}
	responder := newInteractionResponder(session, &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		ID: "interaction", Token: "token",
	}})
	response := ephemeralTextResponse("Olá")
	if err := responder.Respond(response); err != nil {
		t.Fatal(err)
	}
	if !responder.responded() || !strings.Contains(body, "Olá") {
		t.Fatalf("responded = %v, body = %q", responder.responded(), body)
	}
	if err := responder.Respond(response); err == nil {
		t.Fatal("second response accepted")
	}
}

func TestInteractionResponderRejectsNilAndPreservesFailure(t *testing.T) {
	session, err := discordgo.New("Bot token")
	if err != nil {
		t.Fatal(err)
	}
	responder := newInteractionResponder(session, &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{}})
	if err := responder.Respond(nil); err == nil || responder.responded() {
		t.Fatalf("nil response error = %v, responded = %v", err, responder.responded())
	}
	want := errors.New("network")
	session.Client = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, want
	})}
	if err := responder.Respond(ephemeralTextResponse("Olá")); !errors.Is(err, want) || responder.responded() {
		t.Fatalf("network error = %v, responded = %v", err, responder.responded())
	}
}

func TestEphemeralTextResponseUsesComponentsV2AndPtBR(t *testing.T) {
	response := ephemeralTextResponse(ptbr.GenericInteractionError)
	if response.Type != discordgo.InteractionResponseChannelMessageWithSource || response.Data.Flags&discordgo.MessageFlagsEphemeral == 0 || response.Data.Flags&discordgo.MessageFlagsIsComponentsV2 == 0 {
		t.Fatalf("response = %#v", response)
	}
	if got := response.Data.Components[0].(discordgo.TextDisplay).Content; got != ptbr.GenericInteractionError {
		t.Fatalf("content = %q", got)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

// TestDeferredCompletion verifies transport editing and one initial response.
func TestDeferredCompletion(t *testing.T) {
	session, _ := discordgo.New("Bot token")
	edits := 0
	session.Client = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		body := ""
		status := 204
		if r.Method == "PATCH" {
			edits++
			body = `{"id":"1","content":"done"}`
			status = 200
		}
		return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header), Request: r}, nil
	})}
	responder := newInteractionResponder(session, &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{ID: "1", Token: "token", AppID: "app"}})
	if err := responder.Complete(nil); err == nil {
		t.Fatal("unexpected completion")
	}
	if err := responder.Respond(&discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredChannelMessageWithSource}); err != nil {
		t.Fatal(err)
	}
	if err := responder.Complete(&discordgo.InteractionResponseData{Content: "done"}); err != nil || edits != 1 || responder.deferred {
		t.Fatal("completion")
	}
	if err := responder.Respond(ephemeralTextResponse("again")); err == nil {
		t.Fatal("second acknowledgment")
	}
}

// TestInteractionFailures verifies error recovery before and after acknowledgment.
func TestInteractionFailures(t *testing.T) {
	for _, mode := range []string{"ignored", "success", "responded", "reject", "deferred", "autocomplete", "transport"} {
		t.Run(mode, func(t *testing.T) {
			routes := NewRoutes()
			calls := 0
			routes.Commands().Slash("test", "Test", func(_ context.Context, r *command.CommandRequest) error {
				switch mode {
				case "success":
					return nil
				case "responded":
					r.Responder.Respond(ephemeralTextResponse("done"))
					return errors.New("late")
				case "deferred":
					r.Responder.Respond(&discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredChannelMessageWithSource})
					return errors.New("after defer")
				case "reject":
					return command.RejectForbidden("rejected")
				default:
					return errors.New("failure")
				}
			})
			routes.Freeze()
			session, _ := discordgo.New("Bot token")
			session.Client = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				calls++
				if mode == "transport" {
					return nil, errors.New("offline")
				}
				status := 204
				body := ""
				if req.Method == "PATCH" {
					status = 200
					body = `{"id":"1"}`
				}
				return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header), Request: req}, nil
			})}
			kind := discordgo.InteractionApplicationCommand
			if mode == "ignored" {
				kind = discordgo.InteractionPing
			}
			if mode == "autocomplete" {
				kind = discordgo.InteractionApplicationCommandAutocomplete
			}
			interaction := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{Type: kind, ID: "1", AppID: "app", Token: "token", Data: discordgo.ApplicationCommandInteractionData{Name: "test"}}}
			handler := interactionHandler{routes: routes, ctx: context.Background(), logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
			handler.handle(session, interaction)
			expected := 1
			if mode == "success" || mode == "ignored" {
				expected = 0
			}
			if mode == "deferred" {
				expected = 2
			}
			if calls != expected {
				t.Fatalf("REST calls %d want %d", calls, expected)
			}
		})
	}
}
