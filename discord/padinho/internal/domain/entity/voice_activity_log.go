package entity

// VoiceActivityLogStatus reports whether Padinho delivered a voice activity embed.
type VoiceActivityLogStatus string

const (
	// VoiceActivityLogSent means Discord accepted the activity embed.
	VoiceActivityLogSent VoiceActivityLogStatus = "SENT"
	// VoiceActivityLogFailed means Padinho could not render or deliver the activity embed.
	VoiceActivityLogFailed VoiceActivityLogStatus = "FAILED"
)

// VoiceActivityLog records one join, leave, or move observed by Padinho.
type VoiceActivityLog struct {
	ID           uint64                 `gorm:"column:id;primaryKey;autoIncrement"`
	GuildID      uint64                 `gorm:"column:guild_id;not null"`
	UserID       uint64                 `gorm:"column:user_id;not null"`
	OldChannelID *uint64                `gorm:"column:old_channel_id"`
	NewChannelID *uint64                `gorm:"column:new_channel_id"`
	LogStatus    VoiceActivityLogStatus `gorm:"column:log_status;size:6;not null"`
	CreatedAt    int64                  `gorm:"column:created_at;autoCreateTime:milli"`
}

// TableName returns the table storing voice activity delivery records.
func (VoiceActivityLog) TableName() string {
	return "voice_activity_logs"
}
