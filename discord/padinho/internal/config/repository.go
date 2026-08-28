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
	MudaeBotID             = "bots.mudae.id"
	MudaeOCBlueEmojiID     = "bots.mudae.oc.emoji.blue"
	MudaeOCTealEmojiID     = "bots.mudae.oc.emoji.teal"
	MudaeOCGreenEmojiID    = "bots.mudae.oc.emoji.green"
	MudaeOCYellowEmojiID   = "bots.mudae.oc.emoji.yellow"
	MudaeOCOrangeEmojiID   = "bots.mudae.oc.emoji.orange"
	MudaeOCRedEmojiID      = "bots.mudae.oc.emoji.red"
	MudaeOQPurpleEmojiID   = "bots.mudae.oq.emoji.purple"
)

var ErrNotFound = errors.New("configuration value not found")

// MudaeOCSettings contains the bot and custom-emoji IDs required by the $oc listener.
type MudaeOCSettings struct {
	BotID         string
	BlueEmojiID   string
	TealEmojiID   string
	GreenEmojiID  string
	YellowEmojiID string
	OrangeEmojiID string
	RedEmojiID    string
}

// MudaeOQSettings contains the shared sphere and purple emoji IDs required by $oq.
type MudaeOQSettings struct {
	MudaeOCSettings
	PurpleEmojiID string
}

type entry struct {
	Name  string `gorm:"column:name;primaryKey;size:255"`
	Value string `gorm:"column:value;not null"`
}

// MudaeOQ returns the complete required configuration for the $oq listener.
func (r *Repository) MudaeOQ() (MudaeOQSettings, error) {
	shared, err := r.MudaeOC()
	if err != nil {
		return MudaeOQSettings{}, err
	}
	purple, err := r.Get(MudaeOQPurpleEmojiID)
	if err != nil {
		return MudaeOQSettings{}, err
	}
	return MudaeOQSettings{MudaeOCSettings: shared, PurpleEmojiID: purple}, nil
}

// BirthdayDefaultMessage returns the template used when a birthday has no
// custom message.
func (r *Repository) BirthdayDefaultMessage() (string, error) {
	return r.Get(BirthdayDefaultMessage)
}

// MudaeOC returns the complete required configuration for the $oc listener.
func (r *Repository) MudaeOC() (MudaeOCSettings, error) {
	settings := MudaeOCSettings{}
	values := []struct {
		name        string
		destination *string
	}{
		{MudaeBotID, &settings.BotID},
		{MudaeOCBlueEmojiID, &settings.BlueEmojiID},
		{MudaeOCTealEmojiID, &settings.TealEmojiID},
		{MudaeOCGreenEmojiID, &settings.GreenEmojiID},
		{MudaeOCYellowEmojiID, &settings.YellowEmojiID},
		{MudaeOCOrangeEmojiID, &settings.OrangeEmojiID},
		{MudaeOCRedEmojiID, &settings.RedEmojiID},
	}
	for _, value := range values {
		loaded, err := r.Get(value.name)
		if err != nil {
			return MudaeOCSettings{}, err
		}
		*value.destination = loaded
	}
	return settings, nil
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
