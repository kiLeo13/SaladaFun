package mysql

import (
	"errors"

	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
	"gorm.io/gorm"
)

// QuoteRepository selects enabled quotes and their authors with GORM.
type QuoteRepository struct {
	db *gorm.DB
}

// NewQuoteRepository creates a repository backed by the supplied database.
func NewQuoteRepository(db *gorm.DB) *QuoteRepository {
	return &QuoteRepository{db: db}
}

// Random returns one uniformly random enabled quote, or nil when none exist.
func (r *QuoteRepository) Random() (*entity.Quote, error) {
	var quote entity.Quote
	err := r.db.Where("enabled = ?", true).Order("RAND()").Preload("Author").Take(&quote).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &quote, nil
}
