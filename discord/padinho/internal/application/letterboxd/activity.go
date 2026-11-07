package letterboxd

import (
	"context"
	"errors"

	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
)

const HistoryPageSize = 3

var ErrEmptyWishlist = errors.New("no unwatched wishlist titles")

// WatchState supplies the cached title and the actor's current opinion.
type WatchState struct {
	Title     entity.LetterboxdTitle
	Rating    int
	Review    *string
	WatchedAt int64
}

// HistoryEntry is one watched title with its current opinion, never old revisions.
type HistoryEntry struct {
	ID             uint64
	TitleID        uint64
	OriginalTitle  string
	LocalizedTitle string
	MediaType      string
	TMDBID         uint64
	ReleaseYear    int
	WatchedAt      int64
	Rating         int
	Review         *string
}

// ActivityStore persists independently owned opinions and local selection queries.
type ActivityStore interface {
	Watch(context.Context, uint64, uint64) (*WatchState, error)
	SaveWatch(context.Context, uint64, uint64, uint64, int, string) error
	History(context.Context, uint64, string, string, int) ([]HistoryEntry, error)
	RandomUnwatched(context.Context, uint64) (*entity.LetterboxdTitle, error)
}

// Activity applies validation consistently to command, button and modal callers.
type Activity struct{ Store ActivityStore }

// State reads only cached data so opening a modal never waits for TMDB.
func (a *Activity) State(ctx context.Context, user, title uint64) (*WatchState, error) {
	if user == 0 || title == 0 {
		return nil, ErrInvalid
	}
	return a.Store.Watch(ctx, user, title)
}

// Save atomically records a required rating and optional review for the actor.
func (a *Activity) Save(ctx context.Context, user, title, interaction uint64, rating int, text string) error {
	if user == 0 || title == 0 || interaction == 0 || rating < 1 || rating > 5 {
		return ErrInvalid
	}
	if _, err := ReviewText(text); err != nil {
		return err
	}
	return a.Store.SaveWatch(ctx, user, title, interaction, rating, text)
}

// History validates SQL ordering inputs and bounds page offsets.
func (a *Activity) History(ctx context.Context, user uint64, sort, order string, page int) ([]HistoryEntry, error) {
	if user == 0 || page < 1 || page > 100000 || (sort != "date" && sort != "rating") || (order != "asc" && order != "desc") {
		return nil, ErrInvalid
	}
	return a.Store.History(ctx, user, sort, order, page)
}

// Random selects only the actor's unwatched interests without a provider request.
func (a *Activity) Random(ctx context.Context, user uint64) (*entity.LetterboxdTitle, error) {
	if user == 0 {
		return nil, ErrInvalid
	}
	title, err := a.Store.RandomUnwatched(ctx, user)
	if err != nil {
		return nil, err
	}
	if title == nil {
		return nil, ErrEmptyWishlist
	}
	return title, nil
}
