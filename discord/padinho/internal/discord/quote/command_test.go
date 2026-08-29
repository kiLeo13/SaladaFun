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
	Register(registry, &fakeService{random: &entity.Quote{
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
	err := handle(context.Background(), &messagecommand.Request{Responder: responder}, &fakeService{random: &entity.Quote{
		OriginalQuote: "Linha um\nLinha dois", Author: entity.QuoteAuthor{DiscordUserID: &userID},
	}})
	if err != nil || responder.content != "> Linha um\n> Linha dois\n— <@123456789012345678>" || !responder.userMentions {
		t.Fatalf("handle() = %v, %#v", err, responder)
	}
}

func TestQuoteCommandIgnoresInvalidIDsAndEmptyCatalog(t *testing.T) {
	for _, arguments := range [][]string{{"not-a-number"}, {"0"}, {"-1"}} {
		responder := &fakeResponder{}
		service := &fakeService{random: &entity.Quote{OriginalQuote: "Random", Author: entity.QuoteAuthor{Name: "Author"}}}
		if err := handle(context.Background(), &messagecommand.Request{Arguments: arguments, Responder: responder}, service); err != nil || responder.content == "" || service.requestedID != 0 {
			t.Fatalf("invalid ID %q = %v, %#v", arguments, err, responder)
		}
	}
	responder := &fakeResponder{}
	if err := handle(context.Background(), &messagecommand.Request{Responder: responder}, &fakeService{randomErr: appquote.ErrNoQuotes}); err != nil || responder.content == "" {
		t.Fatalf("empty catalog = %v, %#v", err, responder)
	}
}

func TestQuoteCommandFindsDisabledQuoteByValidIDAndReportsMissingID(t *testing.T) {
	responder := &fakeResponder{}
	service := &fakeService{byID: &entity.Quote{ID: 45, OriginalQuote: "Disabled", Enabled: false, Author: entity.QuoteAuthor{Name: "Author"}}}
	if err := handle(context.Background(), &messagecommand.Request{Arguments: []string{"45", "ignored"}, Responder: responder}, service); err != nil || service.requestedID != 45 || responder.content != "> Disabled\n— Author" {
		t.Fatalf("valid ID = %v, %#v, %#v", err, responder, service)
	}
	responder = &fakeResponder{}
	if err := handle(context.Background(), &messagecommand.Request{Arguments: []string{"46"}, Responder: responder}, &fakeService{findErr: appquote.ErrQuoteNotFound}); err != nil || responder.content == "" {
		t.Fatalf("missing ID = %v, %#v", err, responder)
	}
}

func TestQuoteCommandPropagatesFailuresAndRejectsUnsupportedMentionResponder(t *testing.T) {
	want := errors.New("database unavailable")
	if err := handle(context.Background(), &messagecommand.Request{Responder: &fakeResponder{}}, &fakeService{randomErr: want}); !errors.Is(err, want) {
		t.Fatalf("database failure = %v", err)
	}
	userID := uint64(1)
	if err := handle(context.Background(), &messagecommand.Request{Responder: replyOnlyResponder{}}, &fakeService{random: &entity.Quote{
		OriginalQuote: "Test", Author: entity.QuoteAuthor{DiscordUserID: &userID},
	}}); err == nil {
		t.Fatal("unsupported mention responder error = nil")
	}
}

type fakeService struct {
	random      *entity.Quote
	randomErr   error
	byID        *entity.Quote
	findErr     error
	requestedID uint64
}

// Random returns the configured fake selection result.
func (s *fakeService) Random() (*entity.Quote, error) {
	return s.random, s.randomErr
}

// FindByID records the requested ID and returns the configured fake result.
func (s *fakeService) FindByID(id uint64) (*entity.Quote, error) {
	s.requestedID = id
	return s.byID, s.findErr
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
