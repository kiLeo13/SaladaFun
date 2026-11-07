package letterboxd

import (
	"context"
	"errors"
	"testing"

	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
)

type activities struct {
	err    error
	title  *entity.LetterboxdTitle
	writes int
}

// Watch supplies isolated personal state.
func (a *activities) Watch(context.Context, uint64, uint64) (*WatchState, error) {
	return &WatchState{}, a.err
}

// SaveWatch counts validated mutations.
func (a *activities) SaveWatch(context.Context, uint64, uint64, uint64, int, string) error {
	a.writes++
	return a.err
}

// History returns the injected persistence error.
func (a *activities) History(context.Context, uint64, string, string, int) ([]HistoryEntry, error) {
	return nil, a.err
}

// RandomUnwatched returns only the injected eligible candidate.
func (a *activities) RandomUnwatched(context.Context, uint64) (*entity.LetterboxdTitle, error) {
	return a.title, a.err
}

// TestActivityValidation covers all validation and empty/error forwarding paths.
func TestActivityValidation(t *testing.T) {
	ctx := context.Background()
	store := &activities{}
	a := &Activity{store}
	for _, v := range [][2]uint64{{0, 1}, {1, 0}} {
		if _, err := a.State(ctx, v[0], v[1]); err != ErrInvalid {
			t.Fatal("state validation")
		}
	}
	if _, err := a.State(ctx, 1, 1); err != nil {
		t.Fatal(err)
	}
	for _, v := range []struct {
		u, t, i uint64
		r       int
		text    string
	}{{0, 1, 1, 1, ""}, {1, 0, 1, 1, ""}, {1, 1, 0, 1, ""}, {1, 1, 1, 0, ""}, {1, 1, 1, 6, ""}, {1, 1, 1, 1, "x"}} {
		if a.Save(ctx, v.u, v.t, v.i, v.r, v.text) == nil {
			t.Fatal("save validation")
		}
	}
	if err := a.Save(ctx, 1, 1, 1, 5, "Good"); err != nil || store.writes != 1 {
		t.Fatal("save")
	}
	for _, v := range []struct {
		u    uint64
		s, o string
		p    int
	}{{0, "date", "desc", 1}, {1, "bad", "desc", 1}, {1, "date", "bad", 1}, {1, "date", "desc", 0}, {1, "date", "desc", 100001}} {
		if _, err := a.History(ctx, v.u, v.s, v.o, v.p); err != ErrInvalid {
			t.Fatal("history validation")
		}
	}
	if _, err := a.History(ctx, 1, "rating", "asc", 1); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Random(ctx, 0); err != ErrInvalid {
		t.Fatal("random user")
	}
	if _, err := a.Random(ctx, 1); err != ErrEmptyWishlist {
		t.Fatal("empty")
	}
	store.err = errors.New("offline")
	if _, err := a.Random(ctx, 1); err != store.err {
		t.Fatal("forward error")
	}
	store.err = nil
	store.title = &entity.LetterboxdTitle{ID: 1}
	if title, err := a.Random(ctx, 1); err != nil || title.ID != 1 {
		t.Fatal("candidate")
	}
}
