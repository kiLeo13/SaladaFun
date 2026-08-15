package birthday

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/locale/ptbr"
)

func TestMonth(t *testing.T) {
	want := []*entity.Birthday{{UserID: 1}}
	repository := &fakeRepository{monthBirthdays: want}
	service := NewService(repository)
	got, err := service.Month(time.January)
	if err != nil || !reflect.DeepEqual(got, want) || repository.month != time.January {
		t.Fatalf("Month() = %#v, %v", got, err)
	}
	if _, err := service.Month(0); !errors.Is(err, ErrInvalidMonth) {
		t.Fatalf("Month(0) error = %v", err)
	}
}

func TestSaveNormalizesAndDefaults(t *testing.T) {
	repository := &fakeRepository{}
	service := NewService(repository)
	err := service.Save(SaveInput{
		UserID: 1, Name: " Leo ", Birthday: time.Date(2000, 3, 4, 15, 0, 0, 0, time.Local),
		TimeZone: " America/Sao_Paulo ",
	})
	if err != nil {
		t.Fatal(err)
	}
	wantDate := time.Date(2000, 3, 4, 0, 0, 0, 0, time.UTC)
	if repository.saved == nil || repository.saved.Name != "Leo" || repository.saved.TimeZone != "America/Sao_Paulo" || repository.saved.Message != ptbr.BirthdayDefaultMessage || !repository.saved.Birthday.Equal(wantDate) {
		t.Fatalf("saved birthday = %#v", repository.saved)
	}
}

func TestSaveValidation(t *testing.T) {
	valid := SaveInput{
		UserID: 1, Name: "Leo", Birthday: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
		TimeZone: "UTC", Message: "Parabéns, {name}! {mention} faz {age} anos.",
	}
	tests := map[string]struct {
		change func(*SaveInput)
		want   error
	}{
		"user":         {func(input *SaveInput) { input.UserID = 0 }, ErrInvalidUserID},
		"empty name":   {func(input *SaveInput) { input.Name = " " }, ErrInvalidName},
		"long name":    {func(input *SaveInput) { input.Name = string(make([]rune, maximumNameLength+1)) }, ErrInvalidName},
		"zero date":    {func(input *SaveInput) { input.Birthday = time.Time{} }, ErrInvalidDate},
		"future date":  {func(input *SaveInput) { input.Birthday = time.Date(2200, 1, 1, 0, 0, 0, 0, time.UTC) }, ErrInvalidDate},
		"time zone":    {func(input *SaveInput) { input.TimeZone = "Mars/Olympus" }, ErrInvalidTimeZone},
		"long message": {func(input *SaveInput) { input.Message = string(make([]byte, maximumMessageLength+1)) }, ErrInvalidMessage},
		"placeholder":  {func(input *SaveInput) { input.Message = "Olá, {unknown}" }, ErrInvalidMessage},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			input := valid
			test.change(&input)
			if err := NewService(&fakeRepository{}).Save(input); !errors.Is(err, test.want) {
				t.Fatalf("Save() error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestSaveReturnsRepositoryError(t *testing.T) {
	want := errors.New("save")
	repository := &fakeRepository{saveErr: want}
	err := NewService(repository).Save(SaveInput{
		UserID: 1, Name: "Leo", Birthday: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
		TimeZone: "UTC", Message: "Olá",
	})
	if !errors.Is(err, want) {
		t.Fatalf("Save() error = %v", err)
	}
}

func TestDueUsesEachUsersLocalDateAndLedger(t *testing.T) {
	repository := &fakeRepository{birthdays: []*entity.Birthday{
		{UserID: 1, Name: "Leo", Birthday: time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC), TimeZone: "Asia/Tokyo", Message: "Olá"},
		{UserID: 2, Name: "Ana", Birthday: time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC), TimeZone: "UTC", Message: "Olá"},
		{UserID: 3, Name: "Bia", Birthday: time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC), TimeZone: "Asia/Tokyo", Message: "Olá"},
	}, announced: map[uint64]bool{3: true}}
	service := NewService(repository)
	announcements, err := service.Due(time.Date(2026, 1, 1, 15, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if len(announcements) != 1 || announcements[0].UserID != 1 || announcements[0].Age != 26 || announcements[0].LocalDate.Day() != 2 {
		t.Fatalf("Due() = %#v", announcements)
	}
}

func TestDueCelebratesLeapBirthdayOnFebruaryTwentyEighth(t *testing.T) {
	repository := &fakeRepository{birthdays: []*entity.Birthday{{
		UserID: 1, Birthday: time.Date(2000, 2, 29, 0, 0, 0, 0, time.UTC),
		TimeZone: "UTC", Message: "Olá",
	}}}
	announcements, err := NewService(repository).Due(time.Date(2025, 2, 28, 12, 0, 0, 0, time.UTC))
	if err != nil || len(announcements) != 1 || announcements[0].Age != 25 {
		t.Fatalf("Due() = %#v, %v", announcements, err)
	}
	announcements, err = NewService(repository).Due(time.Date(2024, 2, 28, 12, 0, 0, 0, time.UTC))
	if err != nil || len(announcements) != 0 {
		t.Fatalf("leap-year Due() = %#v, %v", announcements, err)
	}
}

func TestDueErrors(t *testing.T) {
	want := errors.New("failure")
	if _, err := NewService(&fakeRepository{listErr: want}).Due(time.Now()); !errors.Is(err, want) {
		t.Fatalf("list error = %v", err)
	}
	invalidZone := &fakeRepository{birthdays: []*entity.Birthday{{UserID: 1, TimeZone: "bad"}}}
	if _, err := NewService(invalidZone).Due(time.Now()); err == nil {
		t.Fatal("invalid stored time zone accepted")
	}
	ledgerFailure := &fakeRepository{
		birthdays:       []*entity.Birthday{{UserID: 1, Birthday: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), TimeZone: "UTC"}},
		announcementErr: want,
	}
	if _, err := NewService(ledgerFailure).Due(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)); !errors.Is(err, want) {
		t.Fatalf("ledger error = %v", err)
	}
}

func TestMarkAnnounced(t *testing.T) {
	repository := &fakeRepository{}
	value := time.Date(2026, 1, 2, 13, 0, 0, 0, time.FixedZone("test", 3600))
	if err := NewService(repository).MarkAnnounced(7, value); err != nil {
		t.Fatal(err)
	}
	if repository.markedUser != 7 || !repository.markedDate.Equal(time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("marked = %d, %v", repository.markedUser, repository.markedDate)
	}
}

type fakeRepository struct {
	monthBirthdays  []*entity.Birthday
	birthdays       []*entity.Birthday
	announced       map[uint64]bool
	saved           *entity.Birthday
	month           time.Month
	markedUser      uint64
	markedDate      time.Time
	saveErr         error
	listErr         error
	announcementErr error
}

func (r *fakeRepository) ListByMonth(month time.Month) ([]*entity.Birthday, error) {
	r.month = month
	return r.monthBirthdays, r.listErr
}

func (r *fakeRepository) List() ([]*entity.Birthday, error) {
	return r.birthdays, r.listErr
}

func (r *fakeRepository) Save(birthday *entity.Birthday) error {
	r.saved = birthday
	return r.saveErr
}

func (r *fakeRepository) WasAnnounced(userID uint64, _ time.Time) (bool, error) {
	return r.announced[userID], r.announcementErr
}

func (r *fakeRepository) MarkAnnounced(userID uint64, localDate time.Time) error {
	r.markedUser, r.markedDate = userID, localDate
	return r.announcementErr
}
