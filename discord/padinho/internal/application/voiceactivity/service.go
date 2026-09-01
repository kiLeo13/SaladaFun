package voiceactivity

import (
	"errors"
	"time"

	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
)

// RecordInput describes one completed voice activity delivery attempt.
type RecordInput struct {
	GuildID      uint64
	UserID       uint64
	OldChannelID *uint64
	NewChannelID *uint64
	Status       entity.VoiceActivityLogStatus
	OccurredAt   time.Time
}

// Service validates and records completed voice activity delivery attempts.
type Service struct {
	repository Repository
}

// NewService constructs a voice activity recording service.
func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

// Record persists one voice activity delivery attempt.
func (s *Service) Record(input RecordInput) error {
	if s.repository == nil {
		return errors.New("voice activity repository is nil")
	}
	if input.GuildID == 0 || input.UserID == 0 {
		return errors.New("voice activity guild and user IDs are required")
	}
	if input.OldChannelID == nil && input.NewChannelID == nil {
		return errors.New("voice activity requires an old or new channel ID")
	}
	if input.OldChannelID != nil && input.NewChannelID != nil && *input.OldChannelID == *input.NewChannelID {
		return errors.New("voice activity channels must differ")
	}
	if input.Status != entity.VoiceActivityLogSent && input.Status != entity.VoiceActivityLogFailed {
		return errors.New("voice activity status is invalid")
	}
	if input.OccurredAt.IsZero() {
		return errors.New("voice activity occurrence time is required")
	}
	return s.repository.Create(&entity.VoiceActivityLog{
		GuildID: input.GuildID, UserID: input.UserID,
		OldChannelID: input.OldChannelID, NewChannelID: input.NewChannelID,
		LogStatus: input.Status, CreatedAt: input.OccurredAt.UTC().UnixMilli(),
	})
}
