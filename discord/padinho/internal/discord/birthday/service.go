package birthday

import (
	"time"

	appbirthday "github.com/kiLeo13/SaladaFun/discord/padinho/internal/application/birthday"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
)

// Service is the application capability consumed by birthday interactions.
type Service interface {
	Month(time.Month) ([]*entity.Birthday, error)
	Next(time.Time) (*appbirthday.UpcomingBirthday, error)
	Birthday(uint64) (*entity.Birthday, error)
	Save(appbirthday.SaveInput) error
	Update(appbirthday.UpdateInput) (*entity.Birthday, error)
}
