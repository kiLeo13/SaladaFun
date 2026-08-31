package entity

// DiscordAccountLink stores one sparse direct-parent relationship between
// Discord accounts.
type DiscordAccountLink struct {
	ID        uint64  `gorm:"column:id;primaryKey;autoIncrement"`
	UserID    uint64  `gorm:"column:user_id;uniqueIndex;not null"`
	ParentID  *uint64 `gorm:"column:parent_id"`
	CreatedAt int64   `gorm:"column:created_at;autoCreateTime:milli"`
	UpdatedAt int64   `gorm:"column:updated_at;autoUpdateTime:milli"`
}

// TableName returns the sparse Discord account hierarchy table.
func (DiscordAccountLink) TableName() string {
	return "discord_account_links"
}
