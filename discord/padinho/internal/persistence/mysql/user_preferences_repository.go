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
	now := time.Now().UTC().UnixMilli()
	firstValue := !defaultValue
	query := "INSERT INTO users_preferences (user_id, auto_mudae_oc, created_at, updated_at) " +
		"VALUES (?, ?, ?, ?) ON DUPLICATE KEY UPDATE " +
		"auto_mudae_oc = NOT COALESCE(auto_mudae_oc, ?), updated_at = VALUES(updated_at)"
	result := r.db.Exec(query, userID, firstValue, now, now, defaultValue)
	if result.Error != nil {
		return false, fmt.Errorf("toggle user preference: %w", result.Error)
	}
	preferences, err := r.FindUserPreferences(userID)
	if err != nil {
		return false, err
	}
	if preferences == nil || preferences.AutoMudaeOC == nil {
		return false, errors.New("toggled user preference was not persisted")
	}
	return *preferences.AutoMudaeOC, nil
}
