package quote

import (
	"errors"

	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
)

var (
	// ErrNoQuotes indicates that no enabled quote is available to publish.
	ErrNoQuotes = errors.New("no enabled quotes available")
	// ErrQuoteNotFound indicates that no quote exists for a requested ID.
	ErrQuoteNotFound = errors.New("quote not found")
)

// Service selects publishable quotes for Discord features.
type Service struct {
	repository Repository
}

// NewService creates a quote selection service.
func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

// Random returns one enabled quote or ErrNoQuotes when the catalog is empty.
func (s *Service) Random() (*entity.Quote, error) {
	quote, err := s.repository.Random()
	if err != nil {
		return nil, err
	}
	if quote == nil {
		return nil, ErrNoQuotes
	}
	return quote, nil
}

// FindByID returns one quote regardless of its enabled state.
func (s *Service) FindByID(id uint64) (*entity.Quote, error) {
	quote, err := s.repository.FindByID(id)
	if err != nil {
		return nil, err
	}
	if quote == nil {
		return nil, ErrQuoteNotFound
	}
	return quote, nil
}
