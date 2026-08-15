package discord

import (
	"errors"
	"io"
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
