package birthday

import (
	"errors"
	"fmt"
	"time"

	appbirthday "github.com/kiLeo13/SaladaFun/discord/padinho/internal/application/birthday"
)

// Service is the application capability consumed by the recurring job.
type Service interface {
	Due(time.Time) ([]appbirthday.Announcement, error)
	MarkAnnounced(uint64, time.Time) error
}

// Sender delivers one rendered birthday announcement.
type Sender interface {
	Send(appbirthday.Announcement) error
}

// Job checks every user's local calendar date and delivers due announcements.
type Job struct {
	service Service
	sender  Sender
	now     func() time.Time
}

func New(service Service, sender Sender) *Job {
	return &Job{service: service, sender: sender, now: time.Now}
}

func (j *Job) Run() error {
	announcements, err := j.service.Due(j.now().UTC())
	if err != nil {
		return fmt.Errorf("find due birthdays: %w", err)
	}
	var failures []error
	for _, announcement := range announcements {
		if err := j.sender.Send(announcement); err != nil {
			failures = append(failures, fmt.Errorf("announce birthday for user %d: %w", announcement.UserID, err))
			continue
		}
		if err := j.service.MarkAnnounced(announcement.UserID, announcement.LocalDate); err != nil {
			failures = append(failures, fmt.Errorf("record birthday for user %d: %w", announcement.UserID, err))
		}
	}
	return errors.Join(failures...)
}
