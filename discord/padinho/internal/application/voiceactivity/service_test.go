package voiceactivity

import (
	"errors"
	"testing"
	"time"

	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
)

func TestServiceRecordsValidatedActivity(t *testing.T) {
	repository := &fakeRepository{}
	now := time.Date(2026, time.September, 1, 3, 16, 17, 577000000, time.UTC)
	newChannelID := uint64(3)
	err := NewService(repository).Record(RecordInput{
		GuildID: 1, UserID: 2, NewChannelID: &newChannelID,
		Status: entity.VoiceActivityLogSent, OccurredAt: now,
	})
	if err != nil || repository.log == nil || repository.log.CreatedAt != now.UnixMilli() || repository.log.LogStatus != entity.VoiceActivityLogSent {
		t.Fatalf("Record() = %#v, %v", repository.log, err)
	}
}

func TestServiceRejectsInvalidRecordsAndPropagatesPersistenceErrors(t *testing.T) {
	channelID := uint64(3)
	now := time.Now()
	tests := map[string]struct {
		service *Service
		input   RecordInput
	}{
		"nil repository":    {service: NewService(nil), input: RecordInput{GuildID: 1, UserID: 2, NewChannelID: &channelID, Status: entity.VoiceActivityLogSent, OccurredAt: now}},
		"missing IDs":       {service: NewService(&fakeRepository{}), input: RecordInput{NewChannelID: &channelID, Status: entity.VoiceActivityLogSent, OccurredAt: now}},
		"missing channels":  {service: NewService(&fakeRepository{}), input: RecordInput{GuildID: 1, UserID: 2, Status: entity.VoiceActivityLogSent, OccurredAt: now}},
		"unchanged channel": {service: NewService(&fakeRepository{}), input: RecordInput{GuildID: 1, UserID: 2, OldChannelID: &channelID, NewChannelID: &channelID, Status: entity.VoiceActivityLogSent, OccurredAt: now}},
		"invalid status":    {service: NewService(&fakeRepository{}), input: RecordInput{GuildID: 1, UserID: 2, NewChannelID: &channelID, Status: "UNKNOWN", OccurredAt: now}},
		"missing time":      {service: NewService(&fakeRepository{}), input: RecordInput{GuildID: 1, UserID: 2, NewChannelID: &channelID, Status: entity.VoiceActivityLogSent}},
		"persistence":       {service: NewService(&fakeRepository{err: errors.New("database unavailable")}), input: RecordInput{GuildID: 1, UserID: 2, NewChannelID: &channelID, Status: entity.VoiceActivityLogSent, OccurredAt: now}},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if err := test.service.Record(test.input); err == nil {
				t.Fatal("Record() error = nil")
			}
		})
	}
}

type fakeRepository struct {
	log *entity.VoiceActivityLog
	err error
}

func (r *fakeRepository) Create(log *entity.VoiceActivityLog) error {
	r.log = log
	return r.err
}
