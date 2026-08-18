// Package config provides access to Padinho's database-backed configuration.
package config

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
)

const (
	AppToken               = "app.token"
	BirthdayChannelID      = "birthday.channel_id"
	BirthdayDefaultMessage = "birthday.defaultMessage"
)

var ErrNotFound = errors.New("configuration value not found")

type entry struct {
	Name  string `gorm:"column:name;primaryKey;size:255"`
	Value string `gorm:"column:value;not null"`
}

// BirthdayDefaultMessage returns the template used when a birthday has no
// custom message.
func (r *Repository) BirthdayDefaultMessage() (string, error) {
	return r.Get(BirthdayDefaultMessage)
}

func (entry) TableName() string {
	return "config"
}

// Repository reads application configuration through GORM.
type Repository struct {
	db *gorm.DB
}

// New binds configuration access to db.
func New(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// Get returns the value stored under name.
func (r *Repository) Get(name string) (string, error) {
	var item entry
	err := r.db.Select("value").Where("name = ?", name).Take(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", fmt.Errorf("%w: %s", ErrNotFound, name)
	}
	if err != nil {
		return "", fmt.Errorf("read configuration %s: %w", name, err)
	}
	return item.Value, nil
}
