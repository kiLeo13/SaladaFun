package mysql

import (
	"context"
	"errors"
	"time"

	app "github.com/kiLeo13/SaladaFun/discord/padinho/internal/application/letterboxd"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// LetterboxdRepository stores global metadata and independently owned activity.
type LetterboxdRepository struct{ db *gorm.DB }

// NewLetterboxdRepository binds the repository to an existing GORM connection.
func NewLetterboxdRepository(db *gorm.DB) *LetterboxdRepository { return &LetterboxdRepository{db} }

// FindTitle resolves a complete cache snapshot, returning nil for a cache miss.
func (r *LetterboxdRepository) FindTitle(ctx context.Context, kind string, id uint64) (*entity.LetterboxdTitle, error) {
	var t entity.LetterboxdTitle
	err := r.db.WithContext(ctx).Where("media_type = ? AND tmdb_id = ?", kind, id).Take(&t).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &t, err
}

// SaveTitle atomically replaces metadata without changing identity or creation time.
func (r *LetterboxdRepository) SaveTitle(ctx context.Context, t *entity.LetterboxdTitle) error {
	if err := app.ValidateTitle(t); err != nil {
		return err
	}
	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "media_type"}, {Name: "tmdb_id"}}, DoUpdates: clause.AssignmentColumns([]string{"original_title", "localized_title", "release_year", "release_date", "synopsis", "poster_path", "backdrop_path", "accent", "accent_source", "metadata_fetched_at", "updated_at"})}).Create(t).Error
	if err != nil {
		return err
	}
	stored, err := r.FindTitle(ctx, t.MediaType, t.TMDBID)
	if err == nil {
		*t = *stored
	}
	return err
}

// AddWishlist idempotently records one user's interest.
func (r *LetterboxdRepository) AddWishlist(ctx context.Context, user, title uint64) error {
	if user == 0 || title == 0 {
		return app.ErrInvalid
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&entity.LetterboxdWishlistItem{UserID: user, TitleID: title}).Error
}

// RemoveWishlist removes only the invoking user's membership.
func (r *LetterboxdRepository) RemoveWishlist(ctx context.Context, user, title uint64) (bool, error) {
	result := r.db.WithContext(ctx).Where("user_id = ? AND title_id = ?", user, title).Delete(&entity.LetterboxdWishlistItem{})
	return result.RowsAffected > 0, result.Error
}

// Wishlist returns cached titles without any provider request.
func (r *LetterboxdRepository) Wishlist(ctx context.Context, user uint64) ([]entity.LetterboxdTitle, error) {
	var titles []entity.LetterboxdTitle
	err := r.db.WithContext(ctx).Table("letterboxd_titles AS t").Select("t.*").Joins("JOIN letterboxd_wishlist_items w ON w.title_id=t.id").Where("w.user_id = ?", user).Order("w.created_at, t.id").Scan(&titles).Error
	return titles, err
}

// SaveWatch serializes rating changes and immutable review revisions per member.
// A repeated or older interaction cannot overwrite a newer submission.
func (r *LetterboxdRepository) SaveWatch(ctx context.Context, user, title, interaction uint64, rating int, text string) error {
	if user == 0 || title == 0 || interaction == 0 || rating < 1 || rating > 5 {
		return app.ErrInvalid
	}
	review, err := app.ReviewText(text)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		watch := entity.LetterboxdWatch{UserID: user, TitleID: title, Rating: rating, MutationID: interaction, WatchedAt: time.Now().UnixMilli()}
		insertion := tx.Clauses(clause.OnConflict{DoUpdates: clause.Assignments(map[string]any{"id": gorm.Expr("id")})}).Create(&watch)
		if insertion.Error != nil {
			return insertion.Error
		}
		var stored entity.LetterboxdWatch
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id=? AND title_id=?", user, title).Take(&stored).Error; err != nil {
			return err
		}
		if stored.MutationID > interaction || (stored.MutationID == interaction && insertion.RowsAffected == 0) {
			return nil
		}
		var latest entity.LetterboxdReview
		err := tx.Where("watch_id=?", stored.ID).Order("id DESC").Take(&latest).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		different := (latest.Review == nil) != (review == nil)
		if latest.Review != nil && review != nil {
			different = *latest.Review != *review
		}
		if different {
			if err := tx.Create(&entity.LetterboxdReview{WatchID: stored.ID, Review: review, InteractionID: interaction}).Error; err != nil {
				return err
			}
		}
		return tx.Model(&stored).Updates(map[string]any{"rating": rating, "mutation_id": interaction}).Error
	})
}

// CurrentReview returns the newest revision, including a nil deletion marker.
func (r *LetterboxdRepository) CurrentReview(ctx context.Context, watch uint64) (*entity.LetterboxdReview, error) {
	var revision entity.LetterboxdReview
	err := r.db.WithContext(ctx).Where("watch_id=?", watch).Order("id DESC").Take(&revision).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &revision, err
}

// CachedSearch supplies bounded complete metadata when remote search fails.
func (r *LetterboxdRepository) CachedSearch(ctx context.Context, query string) ([]entity.LetterboxdTitle, error) {
	var titles []entity.LetterboxdTitle
	pattern := "%" + query + "%"
	err := r.db.WithContext(ctx).Where("original_title LIKE ? OR localized_title LIKE ?", pattern, pattern).Order("id").Limit(25).Find(&titles).Error
	return titles, err
}

// Reviews reads one latest revision per rated watch, including withdrawals.
func (r *LetterboxdRepository) Reviews(ctx context.Context, title uint64) ([]app.Opinion, error) {
	var opinions []app.Opinion
	err := r.db.WithContext(ctx).Table("letterboxd_watches w").Select("w.user_id,w.rating,r.review").Joins("LEFT JOIN letterboxd_reviews r ON r.id=(SELECT MAX(v.id) FROM letterboxd_reviews v WHERE v.watch_id=w.id)").Where("w.title_id=?", title).Order("w.id").Scan(&opinions).Error
	return opinions, err
}
