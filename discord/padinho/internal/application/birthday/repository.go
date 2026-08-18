package birthday

import (
	"time"

	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
)

// Repository is the persistence capability consumed by the birthday service.
type Repository interface {
	ListByMonth(time.Month) ([]*entity.Birthday, error)
	List() ([]*entity.Birthday, error)
	FindByUserID(uint64) (*entity.Birthday, error)
	Save(*entity.Birthday) error
	UpdateName(uint64, string) (bool, error)
	UpdateBirthday(uint64, time.Time) (bool, error)
	UpdateTimeZone(uint64, string) (bool, error)
	UpdateMessage(uint64, string) (bool, error)
	WasAnnounced(uint64, time.Time) (bool, error)
	MarkAnnounced(uint64, time.Time) error
}
