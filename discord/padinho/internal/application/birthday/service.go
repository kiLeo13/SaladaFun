package birthday

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	_ "time/tzdata"
	"unicode/utf8"

	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
)

const (
	maximumNameLength    = 100
	maximumMessageLength = 1800
)

var (
	ErrInvalidUserID   = errors.New("invalid birthday user ID")
	ErrInvalidName     = errors.New("invalid birthday name")
	ErrInvalidDate     = errors.New("invalid birthday date")
	ErrInvalidTimeZone = errors.New("invalid birthday time zone")
	ErrInvalidMessage  = errors.New("invalid birthday message")
	ErrInvalidMonth    = errors.New("invalid birthday month")
	placeholderPattern = regexp.MustCompile(`\{[^{}]+}`)
	allowedPlaceholder = map[string]struct{}{
		"{age}": {}, "{name}": {}, "{mention}": {},
	}
)

// SaveInput contains validated-boundary data for one birthday registration.
type SaveInput struct {
	UserID   uint64
	Name     string
	Birthday time.Time
	TimeZone string
	Message  string
}

// Announcement is a birthday notification ready for Discord rendering.
type Announcement struct {
	UserID    uint64
	Name      string
	Age       int
	Message   string
	LocalDate time.Time
}

// Service owns birthday validation, calendar rules, and announcement selection.
type Service struct {
	repository     Repository
	defaultMessage DefaultMessageProvider
}

func NewService(repository Repository, defaultMessage DefaultMessageProvider) *Service {
	return &Service{repository: repository, defaultMessage: defaultMessage}
}

// Month returns birthdays for one calendar month in day/name order.
func (s *Service) Month(month time.Month) ([]*entity.Birthday, error) {
	if month < time.January || month > time.December {
		return nil, ErrInvalidMonth
	}
	return s.repository.ListByMonth(month)
}

// Save validates and stores one user's birthday.
func (s *Service) Save(input SaveInput) error {
	input.Name = strings.TrimSpace(input.Name)
	input.TimeZone = strings.TrimSpace(input.TimeZone)
	input.Message = strings.TrimSpace(input.Message)
	if input.UserID == 0 {
		return ErrInvalidUserID
	}
	if input.Name == "" || utf8.RuneCountInString(input.Name) > maximumNameLength {
		return ErrInvalidName
	}
	if input.Birthday.IsZero() || input.Birthday.After(time.Now().UTC()) {
		return ErrInvalidDate
	}
	if _, err := time.LoadLocation(input.TimeZone); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidTimeZone, err)
	}
	if err := validateMessage(input.Message); err != nil {
		return ErrInvalidMessage
	}
	input.Birthday = date(input.Birthday)
	return s.repository.Save(&entity.Birthday{
		UserID:   input.UserID,
		Name:     input.Name,
		Birthday: input.Birthday,
		TimeZone: input.TimeZone,
		Message:  input.Message,
	})
}

// Due returns birthdays whose local calendar date is today and has not been announced.
func (s *Service) Due(now time.Time) ([]Announcement, error) {
	birthdays, err := s.repository.List()
	if err != nil {
		return nil, err
	}
	result := make([]Announcement, 0)
	var defaultMessage string
	defaultMessageLoaded := false
	for _, current := range birthdays {
		location, loadErr := time.LoadLocation(current.TimeZone)
		if loadErr != nil {
			return nil, fmt.Errorf("load time zone for user %d: %w", current.UserID, loadErr)
		}
		local := now.In(location)
		localDate := date(local)
		if !occursOn(current.Birthday, localDate) {
			continue
		}
		announced, announcementErr := s.repository.WasAnnounced(current.UserID, localDate)
		if announcementErr != nil {
			return nil, announcementErr
		}
		if announced {
			continue
		}
		message := current.Message
		if message == "" {
			if !defaultMessageLoaded {
				defaultMessage, err = s.defaultMessage.BirthdayDefaultMessage()
				if err != nil {
					return nil, fmt.Errorf("read default birthday message: %w", err)
				}
				if validationErr := validateMessage(defaultMessage); validationErr != nil || defaultMessage == "" {
					return nil, ErrInvalidMessage
				}
				defaultMessageLoaded = true
			}
			message = defaultMessage
		}
		result = append(result, Announcement{
			UserID:    current.UserID,
			Name:      current.Name,
			Age:       localDate.Year() - current.Birthday.Year(),
			Message:   message,
			LocalDate: localDate,
		})
	}
	return result, nil
}

// DefaultMessageProvider retrieves the fallback template used for birthdays
// saved without a custom message.
type DefaultMessageProvider interface {
	BirthdayDefaultMessage() (string, error)
}

func validateMessage(message string) error {
	if len(message) > maximumMessageLength || hasUnknownPlaceholder(message) {
		return ErrInvalidMessage
	}
	return nil
}

// MarkAnnounced records a successfully delivered local-date announcement.
func (s *Service) MarkAnnounced(userID uint64, localDate time.Time) error {
	return s.repository.MarkAnnounced(userID, date(localDate))
}

func hasUnknownPlaceholder(message string) bool {
	for _, placeholder := range placeholderPattern.FindAllString(message, -1) {
		if _, ok := allowedPlaceholder[placeholder]; !ok {
			return true
		}
	}
	return false
}

func date(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}

func occursOn(birthday, current time.Time) bool {
	month, day := birthday.Month(), birthday.Day()
	if month == time.February && day == 29 && !isLeap(current.Year()) {
		day = 28
	}
	return current.Month() == month && current.Day() == day
}

func isLeap(year int) bool {
	return year%400 == 0 || year%4 == 0 && year%100 != 0
}
