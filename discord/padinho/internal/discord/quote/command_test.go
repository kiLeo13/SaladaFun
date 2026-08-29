package quote

import (
	"context"
	"errors"
	"testing"

	appquote "github.com/kiLeo13/SaladaFun/discord/padinho/internal/application/quote"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/messagecommand"
)

func TestQuoteCommandPublishesTranslatedQuoteWithNamedAuthor(t *testing.T) {
	translated := "O tempo não foi desperdiçado."
	registry := messagecommand.NewRegistry()
	Register(registry, &fakeService{quote: &entity.Quote{
		OriginalQuote: "Time was not wasted.", TranslatedQuote: &translated,
		Author: entity.QuoteAuthor{Name: "John Lennon"},
	}})
	if err := registry.Freeze(); err != nil {
		t.Fatal(err)
	}
	responder := &fakeResponder{}
	handled, err := registry.Dispatch(context.Background(), &messagecommand.Request{Content: command, Responder: responder})
	if err != nil || !handled || responder.content != "> O tempo não foi desperdiçado.\n— John Lennon" || responder.userMentions {
		t.Fatalf("Dispatch() = %t, %v, %#v", handled, err, responder)
	}
}

func TestQuoteCommandPublishesOriginalQuoteAndMentionsLinkedUser(t *testing.T) {
	userID := uint64(123456789012345678)
	responder := &fakeResponder{}
	err := handle(context.Background(), &messagecommand.Request{Responder: responder}, &fakeService{quote: &entity.Quote{
		OriginalQuote: "Linha um\nLinha dois", Author: entity.QuoteAuthor{DiscordUserID: &userID},
	}})
	if err != nil || responder.content != "> Linha um\n> Linha dois\n— <@123456789012345678>" || !responder.userMentions {
		t.Fatalf("handle() = %v, %#v", err, responder)
	}
}

func TestQuoteCommandRejectsArgumentsAndEmptyCatalog(t *testing.T) {
	for name, request := range map[string]*messagecommand.Request{
		"arguments": {Arguments: []string{"extra"}, Responder: &fakeResponder{}},
		"empty":     {Responder: &fakeResponder{}},
	} {
		t.Run(name, func(t *testing.T) {
			responder := request.Responder.(*fakeResponder)
			service := &fakeService{}
			if name == "empty" {
				service.err = appquote.ErrNoQuotes
			}
			if err := handle(context.Background(), request, service); err != nil || responder.content == "" {
				t.Fatalf("handle() = %v, %#v", err, responder)
			}
		})
	}
}

func TestQuoteCommandPropagatesFailuresAndRejectsUnsupportedMentionResponder(t *testing.T) {
	want := errors.New("database unavailable")
	if err := handle(context.Background(), &messagecommand.Request{Responder: &fakeResponder{}}, &fakeService{err: want}); !errors.Is(err, want) {
		t.Fatalf("database failure = %v", err)
	}
	userID := uint64(1)
	if err := handle(context.Background(), &messagecommand.Request{Responder: replyOnlyResponder{}}, &fakeService{quote: &entity.Quote{
		OriginalQuote: "Test", Author: entity.QuoteAuthor{DiscordUserID: &userID},
	}}); err == nil {
		t.Fatal("unsupported mention responder error = nil")
	}
}

type fakeService struct {
	quote *entity.Quote
	err   error
}

// Random returns the configured fake selection result.
func (s *fakeService) Random() (*entity.Quote, error) {
	return s.quote, s.err
}

type fakeResponder struct {
	content      string
	userMentions bool
}

// Reply records a reply without enabled mentions.
func (r *fakeResponder) Reply(content string) error {
	r.content = content
	return nil
}

// ReplyWithUserMentions records a reply with user-only mentions enabled.
func (r *fakeResponder) ReplyWithUserMentions(content string) error {
	r.content = content
	r.userMentions = true
	return nil
}

type replyOnlyResponder struct{}

// Reply satisfies the ordinary message-command responder contract.
func (replyOnlyResponder) Reply(string) error {
	return nil
}
