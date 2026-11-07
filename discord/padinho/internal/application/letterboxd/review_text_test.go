package letterboxd

import (
	"context"
	"strings"
	"testing"
)

type normalizedReviewStore struct {
	activities
	text string
}

// SaveWatch captures exactly what the application forwards to persistence.
func (s *normalizedReviewStore) SaveWatch(_ context.Context, _, _, _ uint64, _ int, text string) error {
	s.text = text
	return nil
}

// TestReviewNormalization validates normalized lengths and submission forwarding.
func TestReviewNormalization(t *testing.T) {
	for _, tc := range []struct{ raw, want string }{
		{" \r\n Nice\r\n\n  movie\t yes \r ", "Nice movie yes"},
		{strings.Repeat("\r\n", 250), ""}, {"Hello  world", "Hello world"},
	} {
		if got := NormalizeReview(tc.raw); got != tc.want {
			t.Fatalf("%q => %q", tc.raw, got)
		}
		store := &normalizedReviewStore{}
		if err := (&Activity{Store: store}).Save(context.Background(), 1, 1, 1, 5, tc.raw); err != nil || store.text != tc.want {
			t.Fatal("normalized submission", store.text, err)
		}
	}
	if _, err := ReviewText(" \r x \n "); err != ErrInvalid {
		t.Fatal("normalized single character")
	}
	if _, err := ReviewText(strings.Repeat("a", 501)); err != ErrInvalid {
		t.Fatal("maximum length")
	}
	if text, err := ReviewText(" \n" + strings.Repeat("a", 500) + "\r "); err != nil || len(*text) != 500 {
		t.Fatal("normalized boundary", err)
	}
}
