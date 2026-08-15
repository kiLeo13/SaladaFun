package mysql

import (
	"errors"
	"time"

	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// BirthdayRepository persists birthdays and their announcement ledger with GORM.
type BirthdayRepository struct {
	db *gorm.DB
}

func NewBirthdayRepository(db *gorm.DB) *BirthdayRepository {
	return &BirthdayRepository{db: db}
}

func (r *BirthdayRepository) ListByMonth(month time.Month) ([]*entity.Birthday, error) {
	var birthdays []*entity.Birthday
	err := r.db.Where("MONTH(birthday) = ?", int(month)).
		Order("DAY(birthday), name, user_id").Find(&birthdays).Error
	return birthdays, err
}

func (r *BirthdayRepository) List() ([]*entity.Birthday, error) {
	var birthdays []*entity.Birthday
	err := r.db.Order("MONTH(birthday), DAY(birthday), name, user_id").Find(&birthdays).Error
	return birthdays, err
}

func (r *BirthdayRepository) Save(birthday *entity.Birthday) error {
	return r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"name", "birthday", "time_zone", "message", "updated_at",
		}),
	}).Create(birthday).Error
}

func (r *BirthdayRepository) WasAnnounced(userID uint64, localDate time.Time) (bool, error) {
	var announcement entity.BirthdayAnnouncement
	err := r.db.Where("user_id = ? AND birthday_date = ?", userID, localDate).
		Take(&announcement).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	return err == nil, err
}

func (r *BirthdayRepository) MarkAnnounced(userID uint64, localDate time.Time) error {
	return r.db.Clauses(clause.OnConflict{DoNothing: true}).Create(
		&entity.BirthdayAnnouncement{UserID: userID, BirthdayDate: localDate},
	).Error
}
