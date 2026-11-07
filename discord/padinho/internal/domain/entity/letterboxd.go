package entity

import "time"

// LetterboxdTitle caches a complete TMDB movie or series detail response.
type LetterboxdTitle struct {
	ID                uint64 `gorm:"primaryKey"`
	TMDBID            uint64 `gorm:"column:tmdb_id"`
	MediaType         string
	OriginalTitle     string
	LocalizedTitle    string
	ReleaseYear       int
	ReleaseDate       *time.Time
	Creators          []string        `gorm:"serializer:json;type:json;not null"`
	Genres            []string        `gorm:"serializer:json;type:json;not null"`
	ContentRatings    []ContentRating `gorm:"serializer:json;type:json"`
	Homepage          string
	SeasonCount       int
	Synopsis          string
	PosterPath        string
	BackdropPath      string
	Accent            *uint32
	AccentSource      string
	MetadataFetchedAt int64
	CreatedAt         int64 `gorm:"autoCreateTime:milli"`
	UpdatedAt         int64 `gorm:"autoUpdateTime:milli"`
}

// ContentRating preserves a country's certification and optional movie release type.
type ContentRating struct {
	Country       string `json:"country"`
	Certification string `json:"certification"`
	ReleaseType   int    `json:"release_type,omitempty"`
}

// TableName identifies the shared metadata cache.
func (LetterboxdTitle) TableName() string { return "letterboxd_titles" }

// LetterboxdWishlistItem records one member's interest in a title.
type LetterboxdWishlistItem struct {
	UserID    uint64 `gorm:"primaryKey;autoIncrement:false"`
	TitleID   uint64 `gorm:"primaryKey;autoIncrement:false"`
	CreatedAt int64  `gorm:"autoCreateTime:milli"`
}

// TableName identifies personal wishlist membership.
func (LetterboxdWishlistItem) TableName() string { return "letterboxd_wishlist_items" }

// LetterboxdWatch holds one member's current rating and original watch date.
type LetterboxdWatch struct {
	ID         uint64 `gorm:"primaryKey"`
	UserID     uint64
	TitleID    uint64
	WatchedAt  int64
	Rating     int
	MutationID uint64
	CreatedAt  int64 `gorm:"autoCreateTime:milli"`
	UpdatedAt  int64 `gorm:"autoUpdateTime:milli"`
}

// TableName identifies individual watch state.
func (LetterboxdWatch) TableName() string { return "letterboxd_watches" }

// LetterboxdReview is an immutable revision; nil Review withdraws previous text.
type LetterboxdReview struct {
	ID            uint64 `gorm:"primaryKey"`
	WatchID       uint64
	Review        *string
	InteractionID uint64
	CreatedAt     int64 `gorm:"autoCreateTime:milli"`
}

// TableName identifies the append-only review ledger.
func (LetterboxdReview) TableName() string { return "letterboxd_reviews" }
