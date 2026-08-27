package mysql

import (
	"errors"
	"fmt"
	"time"

	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
	"gorm.io/gorm"
)

// UserPreferencesRepository persists the shared nullable Discord user settings.
type UserPreferencesRepository struct {
	db *gorm.DB
}

// NewUserPreferencesRepository binds user-preference persistence to GORM.
func NewUserPreferencesRepository(db *gorm.DB) *UserPreferencesRepository {
	return &UserPreferencesRepository{db: db}
}

// FindUserPreferences returns nil when the user has never stored a preference.
func (r *UserPreferencesRepository) FindUserPreferences(userID uint64) (*entity.UserPreferences, error) {
	var preferences entity.UserPreferences
	err := r.db.Where("user_id = ?", userID).Take(&preferences).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find user preferences: %w", err)
	}
	return &preferences, nil
}

// ToggleAutoMudaeOC atomically inverts the explicit or effective default setting.
func (r *UserPreferencesRepository) ToggleAutoMudaeOC(userID uint64, defaultValue bool) (bool, error) {
	return r.toggleBoolean(userID, defaultValue, "auto_mudae_oc", func(preferences *entity.UserPreferences) *bool {
		return preferences.AutoMudaeOC
	})
}

// ToggleAutoMudaeOQ atomically inverts the explicit or effective default setting.
func (r *UserPreferencesRepository) ToggleAutoMudaeOQ(userID uint64, defaultValue bool) (bool, error) {
	return r.toggleBoolean(userID, defaultValue, "auto_mudae_oq", func(preferences *entity.UserPreferences) *bool {
		return preferences.AutoMudaeOQ
	})
}

// ToggleAutoMudaeOH atomically inverts the explicit or effective default setting.
func (r *UserPreferencesRepository) ToggleAutoMudaeOH(userID uint64, defaultValue bool) (bool, error) {
	return r.toggleBoolean(userID, defaultValue, "auto_mudae_oh", func(preferences *entity.UserPreferences) *bool {
		return preferences.AutoMudaeOH
	})
}

// toggleBoolean performs the shared nullable-boolean MySQL upsert operation.
func (r *UserPreferencesRepository) toggleBoolean(userID uint64, defaultValue bool, column string, selected func(*entity.UserPreferences) *bool) (bool, error) {
	now := time.Now().UTC().UnixMilli()
	firstValue := !defaultValue
	query := "INSERT INTO users_preferences (user_id, " + column + ", created_at, updated_at) " +
		"VALUES (?, ?, ?, ?) ON DUPLICATE KEY UPDATE " +
		column + " = NOT COALESCE(" + column + ", ?), updated_at = VALUES(updated_at)"
	result := r.db.Exec(query, userID, firstValue, now, now, defaultValue)
	if result.Error != nil {
		return false, fmt.Errorf("toggle user preference: %w", result.Error)
	}
	preferences, err := r.FindUserPreferences(userID)
	if err != nil {
		return false, err
	}
	if preferences == nil || selected(preferences) == nil {
		return false, errors.New("toggled user preference was not persisted")
	}
	return *selected(preferences), nil
}
