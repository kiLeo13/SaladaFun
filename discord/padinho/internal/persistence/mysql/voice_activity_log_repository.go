package mysql

import (
	"fmt"

	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
	"gorm.io/gorm"
)

// VoiceActivityLogRepository persists voice activity delivery results with GORM.
type VoiceActivityLogRepository struct {
	db *gorm.DB
}

// NewVoiceActivityLogRepository binds voice activity persistence to GORM.
func NewVoiceActivityLogRepository(db *gorm.DB) *VoiceActivityLogRepository {
	return &VoiceActivityLogRepository{db: db}
}

// Create inserts one completed voice activity delivery record.
func (r *VoiceActivityLogRepository) Create(log *entity.VoiceActivityLog) error {
	if err := r.db.Create(log).Error; err != nil {
		return fmt.Errorf("create voice activity log: %w", err)
	}
	return nil
}
