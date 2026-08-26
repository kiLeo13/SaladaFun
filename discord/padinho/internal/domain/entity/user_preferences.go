package entity

// UserPreferences stores nullable, module-owned preferences for one Discord user.
type UserPreferences struct {
	UserID      uint64 `gorm:"column:user_id;primaryKey;autoIncrement:false"`
	AutoMudaeOC *bool  `gorm:"column:auto_mudae_oc"`
	CreatedAt   int64  `gorm:"column:created_at;autoCreateTime:milli"`
	UpdatedAt   int64  `gorm:"column:updated_at;autoUpdateTime:milli"`
}

// TableName returns the shared generic user-preference table.
func (UserPreferences) TableName() string {
	return "users_preferences"
}
