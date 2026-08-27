// Package preferences resolves nullable persisted user preferences into effective behavior.
package preferences

import (
	"fmt"
	"sync"

	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
)

// DefaultAutoMudaeOC is the effective automatic-assistance setting when no choice exists.
const DefaultAutoMudaeOC = true

// DefaultAutoMudaeOQ is the effective automatic-assistance setting when no choice exists.
const DefaultAutoMudaeOQ = true

// DefaultAutoMudaeOH is the effective automatic-assistance setting when no choice exists.
const DefaultAutoMudaeOH = true

// Repository persists generic user preferences without deciding module defaults.
type Repository interface {
	FindUserPreferences(userID uint64) (*entity.UserPreferences, error)
	ToggleAutoMudaeOC(userID uint64, defaultValue bool) (bool, error)
	ToggleAutoMudaeOQ(userID uint64, defaultValue bool) (bool, error)
	ToggleAutoMudaeOH(userID uint64, defaultValue bool) (bool, error)
}

// AutoMudaeOH reports whether automatic Ouroharvest assistance is enabled.
func (s *Service) AutoMudaeOH(userID uint64) (bool, error) {
	preferences, err := s.repository.FindUserPreferences(userID)
	if err != nil {
		return false, fmt.Errorf("read user preferences: %w", err)
	}
	if preferences == nil || preferences.AutoMudaeOH == nil {
		return DefaultAutoMudaeOH, nil
	}
	return *preferences.AutoMudaeOH, nil
}

// AutoMudaeOQ reports whether automatic Ouroquest assistance is enabled.
func (s *Service) AutoMudaeOQ(userID uint64) (bool, error) {
	preferences, err := s.repository.FindUserPreferences(userID)
	if err != nil {
		return false, fmt.Errorf("read user preferences: %w", err)
	}
	if preferences == nil || preferences.AutoMudaeOQ == nil {
		return DefaultAutoMudaeOQ, nil
	}
	return *preferences.AutoMudaeOQ, nil
}

// Service owns effective preference defaults and mutation behavior.
type Service struct {
	repository Repository
	toggleMu   sync.Mutex
}

// ToggleAutoMudaeOQ atomically inverts the user's effective automatic setting.
func (s *Service) ToggleAutoMudaeOQ(userID uint64) (bool, error) {
	s.toggleMu.Lock()
	defer s.toggleMu.Unlock()
	enabled, err := s.repository.ToggleAutoMudaeOQ(userID, DefaultAutoMudaeOQ)
	if err != nil {
		return false, fmt.Errorf("toggle automatic Mudae Ouroquest preference: %w", err)
	}
	return enabled, nil
}

// ToggleAutoMudaeOH atomically inverts the user's effective automatic setting.
func (s *Service) ToggleAutoMudaeOH(userID uint64) (bool, error) {
	s.toggleMu.Lock()
	defer s.toggleMu.Unlock()
	enabled, err := s.repository.ToggleAutoMudaeOH(userID, DefaultAutoMudaeOH)
	if err != nil {
		return false, fmt.Errorf("toggle automatic Mudae Ouroharvest preference: %w", err)
	}
	return enabled, nil
}

// NewService constructs the user-preference application service.
func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

// AutoMudaeOC reports whether automatic Ourochest assistance is enabled.
func (s *Service) AutoMudaeOC(userID uint64) (bool, error) {
	preferences, err := s.repository.FindUserPreferences(userID)
	if err != nil {
		return false, fmt.Errorf("read user preferences: %w", err)
	}
	if preferences == nil || preferences.AutoMudaeOC == nil {
		return DefaultAutoMudaeOC, nil
	}
	return *preferences.AutoMudaeOC, nil
}

// ToggleAutoMudaeOC atomically inverts the user's effective automatic setting.
func (s *Service) ToggleAutoMudaeOC(userID uint64) (bool, error) {
	s.toggleMu.Lock()
	defer s.toggleMu.Unlock()
	enabled, err := s.repository.ToggleAutoMudaeOC(userID, DefaultAutoMudaeOC)
	if err != nil {
		return false, fmt.Errorf("toggle automatic Mudae Ourochest preference: %w", err)
	}
	return enabled, nil
}
