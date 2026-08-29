package quote

import (
	"errors"
	"testing"

	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
)

func TestServiceRandom(t *testing.T) {
	stored := &entity.Quote{ID: 1}
	service := NewService(&fakeRepository{quote: stored})
	quote, err := service.Random()
	if err != nil || quote != stored {
		t.Fatalf("Random() = %#v, %v", quote, err)
	}
}

func TestServiceRandomRejectsEmptyCatalog(t *testing.T) {
	service := NewService(&fakeRepository{})
	if _, err := service.Random(); !errors.Is(err, ErrNoQuotes) {
		t.Fatalf("Random() error = %v", err)
	}
}

func TestServiceRandomPropagatesRepositoryFailure(t *testing.T) {
	want := errors.New("database unavailable")
	service := NewService(&fakeRepository{err: want})
	if _, err := service.Random(); !errors.Is(err, want) {
		t.Fatalf("Random() error = %v", err)
	}
}

type fakeRepository struct {
	quote *entity.Quote
	err   error
}

// Random returns the configured fake result.
func (r *fakeRepository) Random() (*entity.Quote, error) {
	return r.quote, r.err
}
