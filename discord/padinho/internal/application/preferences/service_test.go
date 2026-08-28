package preferences

import (
	"errors"
	"testing"

	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
)

func TestAutoMudaeOCResolvesMissingNullableAndExplicitValues(t *testing.T) {
	enabled, disabled := true, false
	tests := []struct {
		name        string
		preferences *entity.UserPreferences
		want        bool
	}{
		{"missing", nil, true},
		{"nullable", &entity.UserPreferences{}, true},
		{"enabled", &entity.UserPreferences{AutoMudaeOC: &enabled}, true},
		{"disabled", &entity.UserPreferences{AutoMudaeOC: &disabled}, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := &preferenceRepositoryStub{preferences: test.preferences}
			got, err := NewService(repository).AutoMudaeOC(123)
			if err != nil || got != test.want || repository.userID != 123 {
				t.Fatalf("AutoMudaeOC() = %t, %v; user = %d", got, err, repository.userID)
			}
		})
	}
}

func TestAutoMudaeOQResolvesMissingNullableAndExplicitValues(t *testing.T) {
	enabled, disabled := true, false
	for _, test := range []struct {
		name        string
		preferences *entity.UserPreferences
		want        bool
	}{
		{"missing", nil, true}, {"nullable", &entity.UserPreferences{}, true}, {"enabled", &entity.UserPreferences{AutoMudaeOQ: &enabled}, true}, {"disabled", &entity.UserPreferences{AutoMudaeOQ: &disabled}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			repository := &preferenceRepositoryStub{preferences: test.preferences}
			got, err := NewService(repository).AutoMudaeOQ(123)
			if err != nil || got != test.want {
				t.Fatalf("AutoMudaeOQ() = %t, %v", got, err)
			}
		})
	}
}

func TestPreferenceServiceWrapsRepositoryErrorsAndToggles(t *testing.T) {
	want := errors.New("database unavailable")
	repository := &preferenceRepositoryStub{err: want}
	service := NewService(repository)
	if _, err := service.AutoMudaeOC(1); !errors.Is(err, want) {
		t.Fatalf("AutoMudaeOC() error = %v", err)
	}
	if _, err := service.ToggleAutoMudaeOC(1); !errors.Is(err, want) {
		t.Fatalf("ToggleAutoMudaeOC() error = %v", err)
	}
	if _, err := service.AutoMudaeOQ(1); !errors.Is(err, want) {
		t.Fatalf("AutoMudaeOQ() error = %v", err)
	}
	if _, err := service.ToggleAutoMudaeOQ(1); !errors.Is(err, want) {
		t.Fatalf("ToggleAutoMudaeOQ() error = %v", err)
	}
	repository.err = nil
	repository.toggle = false
	if enabled, err := service.ToggleAutoMudaeOC(9); err != nil || enabled || repository.defaultValue != DefaultAutoMudaeOC {
		t.Fatalf("ToggleAutoMudaeOC() = %t, %v; default = %t", enabled, err, repository.defaultValue)
	}
}

type preferenceRepositoryStub struct {
	preferences  *entity.UserPreferences
	toggle       bool
	err          error
	userID       uint64
	defaultValue bool
}

func (r *preferenceRepositoryStub) FindUserPreferences(userID uint64) (*entity.UserPreferences, error) {
	r.userID = userID
	return r.preferences, r.err
}

func (r *preferenceRepositoryStub) ToggleAutoMudaeOC(userID uint64, defaultValue bool) (bool, error) {
	r.userID = userID
	r.defaultValue = defaultValue
	return r.toggle, r.err
}

func (r *preferenceRepositoryStub) ToggleAutoMudaeOQ(userID uint64, defaultValue bool) (bool, error) {
	return r.ToggleAutoMudaeOC(userID, defaultValue)
}
