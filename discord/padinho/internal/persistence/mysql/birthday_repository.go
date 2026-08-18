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

func (r *BirthdayRepository) FindByUserID(userID uint64) (*entity.Birthday, error) {
	var birthday entity.Birthday
	err := r.db.Where("user_id = ?", userID).Take(&birthday).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &birthday, err
}

func (r *BirthdayRepository) Save(birthday *entity.Birthday) error {
	return r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"name", "birthday", "time_zone", "message", "updated_at",
		}),
	}).Create(birthday).Error
}

func (r *BirthdayRepository) UpdateName(userID uint64, name string) (bool, error) {
	return r.updateColumn(userID, "name", name)
}

func (r *BirthdayRepository) UpdateBirthday(userID uint64, birthday time.Time) (bool, error) {
	return r.updateColumn(userID, "birthday", birthday)
}

func (r *BirthdayRepository) UpdateTimeZone(userID uint64, timeZone string) (bool, error) {
	return r.updateColumn(userID, "time_zone", timeZone)
}

func (r *BirthdayRepository) UpdateMessage(userID uint64, message string) (bool, error) {
	return r.updateColumn(userID, "message", message)
}

func (r *BirthdayRepository) updateColumn(userID uint64, column string, value any) (bool, error) {
	result := r.db.Model(&entity.Birthday{}).Where("user_id = ?", userID).Update(column, value)
	if result.Error != nil || result.RowsAffected > 0 {
		return result.RowsAffected > 0, result.Error
	}
	var count int64
	err := r.db.Model(&entity.Birthday{}).Where("user_id = ?", userID).Count(&count).Error
	return count == 1, err
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
