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
	return r.find(r.db.Where("enabled = ?", true).Order("RAND()"))
}

// FindByID returns one quote by primary key regardless of its enabled state.
func (r *QuoteRepository) FindByID(id uint64) (*entity.Quote, error) {
	return r.find(r.db.Where("id = ?", id))
}

// find loads one quote and its canonical author from a prepared query.
func (r *QuoteRepository) find(query *gorm.DB) (*entity.Quote, error) {
	var quote entity.Quote
	err := query.Preload("Author").Take(&quote).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &quote, nil
}
