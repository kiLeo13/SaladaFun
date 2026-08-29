package quote

import (
	"errors"

	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
)

// ErrNoQuotes indicates that no enabled quote is available to publish.
var ErrNoQuotes = errors.New("no enabled quotes available")

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
