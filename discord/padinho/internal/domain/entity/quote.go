package entity

// QuoteAuthor identifies the canonical author of one or more quotes.
type QuoteAuthor struct {
	ID            uint64  `gorm:"column:id;primaryKey;autoIncrement"`
	Name          string  `gorm:"column:name;size:100;not null"`
	DiscordUserID *uint64 `gorm:"column:discord_user_id"`
	CreatedAt     int64   `gorm:"column:created_at;autoCreateTime:milli"`
	UpdatedAt     int64   `gorm:"column:updated_at;autoUpdateTime:milli"`
}

// TableName returns the table storing canonical quote authors.
func (QuoteAuthor) TableName() string {
	return "quote_authors"
}

// Quote is one attributed statement that Padinho may publish.
type Quote struct {
	ID              uint64      `gorm:"column:id;primaryKey;autoIncrement"`
	AuthorID        uint64      `gorm:"column:author_id;not null"`
	Author          QuoteAuthor `gorm:"foreignKey:AuthorID;references:ID"`
	OriginalQuote   string      `gorm:"column:original_quote;size:1800;not null"`
	TranslatedQuote *string     `gorm:"column:translated_quote;size:1800"`
	SourceURL       *string     `gorm:"column:source_url;size:2048"`
	Enabled         bool        `gorm:"column:enabled;not null"`
	CreatedAt       int64       `gorm:"column:created_at;autoCreateTime:milli"`
	UpdatedAt       int64       `gorm:"column:updated_at;autoUpdateTime:milli"`
}

// TableName returns the table storing attributed quotes.
func (Quote) TableName() string {
	return "quotes"
}
