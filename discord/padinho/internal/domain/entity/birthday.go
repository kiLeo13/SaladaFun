package entity

import "time"

// Birthday is one Discord user's birthday and announcement preferences.
type Birthday struct {
	UserID    uint64    `gorm:"column:user_id;primaryKey;autoIncrement:false"`
	Name      string    `gorm:"column:name;size:100;not null"`
	Birthday  time.Time `gorm:"column:birthday;type:date;not null"`
	TimeZone  string    `gorm:"column:time_zone;size:255;not null"`
	Message   string    `gorm:"column:message;type:text;not null"`
	CreatedAt int64     `gorm:"column:created_at;autoCreateTime:milli"`
	UpdatedAt int64     `gorm:"column:updated_at;autoUpdateTime:milli"`
}

func (Birthday) TableName() string {
	return "birthdays"
}

// BirthdayAnnouncement records one completed local-date announcement.
type BirthdayAnnouncement struct {
	UserID       uint64    `gorm:"column:user_id;primaryKey;autoIncrement:false"`
	BirthdayDate time.Time `gorm:"column:birthday_date;type:date;primaryKey"`
	SentAt       int64     `gorm:"column:sent_at;autoCreateTime:milli"`
}

func (BirthdayAnnouncement) TableName() string {
	return "birthday_announcements"
}
