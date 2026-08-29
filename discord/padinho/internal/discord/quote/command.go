// Package quote exposes Padinho's literal random-quote command.
package quote

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	appquote "github.com/kiLeo13/SaladaFun/discord/padinho/internal/application/quote"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/locale/ptbr"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/messagecommand"
)

const command = "!quote"

// Service selects one enabled quote for publication.
type Service interface {
	Random() (*entity.Quote, error)
	FindByID(uint64) (*entity.Quote, error)
}

// Register declares the random-quote message command.
func Register(registry *messagecommand.Registry, service Service) {
	registry.Command(command, func(ctx context.Context, request *messagecommand.Request) error {
		return handle(ctx, request, service)
	})
}

// handle publishes a random quote in the requested two-line format.
func handle(_ context.Context, request *messagecommand.Request, service Service) error {
	var quote *entity.Quote
	var err error
	if quoteID, valid := requestedID(request.Arguments); valid {
		quote, err = service.FindByID(quoteID)
		if errors.Is(err, appquote.ErrQuoteNotFound) {
			return request.Responder.Reply(ptbr.QuoteNotFound)
		}
	} else {
		quote, err = service.Random()
		if errors.Is(err, appquote.ErrNoQuotes) {
			return request.Responder.Reply(ptbr.QuoteEmpty)
		}
	}
	if err != nil {
		return err
	}
	content := render(quote)
	if quote.Author.DiscordUserID == nil {
		return request.Responder.Reply(content)
	}
	responder, ok := request.Responder.(messagecommand.UserMentionResponder)
	if !ok {
		return errors.New("quote responder does not support user mentions")
	}
	return responder.ReplyWithUserMentions(content)
}

// requestedID returns the positive decimal quote ID in the first argument.
func requestedID(arguments []string) (uint64, bool) {
	if len(arguments) == 0 {
		return 0, false
	}
	id, err := strconv.ParseUint(arguments[0], 10, 64)
	return id, err == nil && id != 0
}

// render formats one quote without rendering its optional provenance URL.
func render(quote *entity.Quote) string {
	text := quote.OriginalQuote
	if quote.TranslatedQuote != nil {
		text = *quote.TranslatedQuote
	}
	text = strings.ReplaceAll(text, "\n", "\n> ")
	author := quote.Author.Name
	if quote.Author.DiscordUserID != nil {
		author = fmt.Sprintf("<@%d>", *quote.Author.DiscordUserID)
	}
	return fmt.Sprintf("> %s\n— %s", text, author)
}
