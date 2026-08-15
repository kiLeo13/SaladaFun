package birthday

import (
	"time"

	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
)

// Repository is the persistence capability consumed by the birthday service.
type Repository interface {
	ListByMonth(time.Month) ([]*entity.Birthday, error)
	List() ([]*entity.Birthday, error)
	Save(*entity.Birthday) error
	WasAnnounced(uint64, time.Time) (bool, error)
	MarkAnnounced(uint64, time.Time) error
}
