// Package configuration provides typed access to Padinho's database-backed
// application configuration.
package configuration

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

const AppTokenName = "app.token"

var ErrNotFound = errors.New("configuration value not found")

type entry struct {
	Name  string `gorm:"column:name;primaryKey;size:255"`
	Value string `gorm:"column:value;not null"`
}

func (entry) TableName() string {
	return "config"
}

// Repository reads application configuration through GORM.
type Repository struct {
	database *gorm.DB
}

// NewRepository binds configuration access to the provided GORM connection.
func NewRepository(database *gorm.DB) *Repository {
	return &Repository{database: database}
}

// Get returns the value stored under name.
func (r *Repository) Get(ctx context.Context, name string) (string, error) {
	var item entry
	err := r.database.WithContext(ctx).
		Select("value").
		Where("name = ?", name).
		Take(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", fmt.Errorf("%w: %s", ErrNotFound, name)
	}
	if err != nil {
		return "", fmt.Errorf("read configuration %s: %w", name, err)
	}
	return item.Value, nil
}
