package birthday

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
)

func TestMonth(t *testing.T) {
	want := []*entity.Birthday{{UserID: 1}}
	repository := &fakeRepository{monthBirthdays: want}
	service := newService(repository)
	got, err := service.Month(time.January)
	if err != nil || !reflect.DeepEqual(got, want) || repository.month != time.January {
		t.Fatalf("Month() = %#v, %v", got, err)
	}
	if _, err := service.Month(0); !errors.Is(err, ErrInvalidMonth) {
		t.Fatalf("Month(0) error = %v", err)
	}
}

func TestSaveNormalizesAndTrimsMessage(t *testing.T) {
	repository := &fakeRepository{}
	service := newService(repository)
	err := service.Save(SaveInput{
		UserID: 1, Name: " Leo ", Birthday: time.Date(2000, 3, 4, 15, 0, 0, 0, time.Local),
		TimeZone: " America/Sao_Paulo ", Message: " \t ",
	})
	if err != nil {
		t.Fatal(err)
	}
	wantDate := time.Date(2000, 3, 4, 0, 0, 0, 0, time.UTC)
	if repository.saved == nil || repository.saved.Name != "Leo" || repository.saved.TimeZone != "America/Sao_Paulo" || repository.saved.Message != "" || !repository.saved.Birthday.Equal(wantDate) {
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
			if err := newService(&fakeRepository{}).Save(input); !errors.Is(err, test.want) {
				t.Fatalf("Save() error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestSaveReturnsRepositoryError(t *testing.T) {
	want := errors.New("save")
	repository := &fakeRepository{saveErr: want}
	err := newService(repository).Save(SaveInput{
		UserID: 1, Name: "Leo", Birthday: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
		TimeZone: "UTC", Message: "Olá",
	})
	if !errors.Is(err, want) {
		t.Fatalf("Save() error = %v", err)
	}
}

func TestBirthdayReturnsRegistrationAndErrors(t *testing.T) {
	want := &entity.Birthday{UserID: 7, Name: "Leo"}
	service := newService(&fakeRepository{found: want})
	if got, err := service.Birthday(7); err != nil || got != want {
		t.Fatalf("Birthday() = %#v, %v", got, err)
	}
	if _, err := service.Birthday(0); !errors.Is(err, ErrInvalidUserID) {
		t.Fatalf("Birthday(0) error = %v", err)
	}
	if _, err := newService(&fakeRepository{}).Birthday(7); !errors.Is(err, ErrBirthdayNotFound) {
		t.Fatalf("missing Birthday() error = %v", err)
	}
	wantErr := errors.New("find")
	if _, err := newService(&fakeRepository{findErr: wantErr}).Birthday(7); !errors.Is(err, wantErr) {
		t.Fatalf("Birthday() repository error = %v", err)
	}
}

func TestUpdateValidatesAndChangesExactlyOneField(t *testing.T) {
	dateValue := time.Date(1999, 4, 5, 13, 0, 0, 0, time.Local)
	nameValue, zoneValue, messageValue := " Leo ", " America/Manaus ", " Olá, {name} "
	tests := map[string]struct {
		input appUpdateInput
		want  any
	}{
		"name":     {appUpdateInput{name: &nameValue}, "Leo"},
		"birthday": {appUpdateInput{birthday: &dateValue}, time.Date(1999, 4, 5, 0, 0, 0, 0, time.UTC)},
		"timezone": {appUpdateInput{timeZone: &zoneValue}, "America/Manaus"},
		"message":  {appUpdateInput{message: &messageValue}, "Olá, {name}"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			repository := &fakeRepository{found: &entity.Birthday{UserID: 7}}
			input := UpdateInput{UserID: 7, Name: test.input.name, Birthday: test.input.birthday, TimeZone: test.input.timeZone, Message: test.input.message}
			got, err := newService(repository).Update(input)
			if err != nil || got != repository.found || !reflect.DeepEqual(repository.updatedValue, test.want) {
				t.Fatalf("Update() = %#v, %v; value = %#v", got, err, repository.updatedValue)
			}
		})
	}
}

type appUpdateInput struct {
	name     *string
	birthday *time.Time
	timeZone *string
	message  *string
}

func TestUpdateRejectsInvalidAndMissingRecords(t *testing.T) {
	name, second := "Leo", "Ana"
	invalid := []struct {
		name  string
		input UpdateInput
		want  error
	}{
		{"user", UpdateInput{Name: &name}, ErrInvalidUserID},
		{"no field", UpdateInput{UserID: 7}, ErrInvalidUpdate},
		{"multiple fields", UpdateInput{UserID: 7, Name: &name, Message: &second}, ErrInvalidUpdate},
		{"name", UpdateInput{UserID: 7, Name: ptrString(" ")}, ErrInvalidName},
		{"date", UpdateInput{UserID: 7, Birthday: &time.Time{}}, ErrInvalidDate},
		{"timezone", UpdateInput{UserID: 7, TimeZone: ptrString("Mars/Olympus")}, ErrInvalidTimeZone},
		{"message", UpdateInput{UserID: 7, Message: ptrString("{unknown}")}, ErrInvalidMessage},
	}
	for _, test := range invalid {
		t.Run(test.name, func(t *testing.T) {
			if _, err := newService(&fakeRepository{}).Update(test.input); !errors.Is(err, test.want) {
				t.Fatalf("Update() error = %v, want %v", err, test.want)
			}
		})
	}
	if _, err := newService(&fakeRepository{updateFound: ptrBool(false)}).Update(UpdateInput{UserID: 7, Name: &name}); !errors.Is(err, ErrBirthdayNotFound) {
		t.Fatalf("missing Update() error = %v", err)
	}
	wantErr := errors.New("update")
	if _, err := newService(&fakeRepository{updateErr: wantErr}).Update(UpdateInput{UserID: 7, Name: &name}); !errors.Is(err, wantErr) {
		t.Fatalf("Update() repository error = %v", err)
	}
}

func ptrString(value string) *string { return &value }
func ptrBool(value bool) *bool       { return &value }

func TestDueUsesEachUsersLocalDateAndLedger(t *testing.T) {
	repository := &fakeRepository{birthdays: []*entity.Birthday{
		{UserID: 1, Name: "Leo", Birthday: time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC), TimeZone: "Asia/Tokyo", Message: "Olá"},
		{UserID: 2, Name: "Ana", Birthday: time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC), TimeZone: "UTC", Message: "Olá"},
		{UserID: 3, Name: "Bia", Birthday: time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC), TimeZone: "Asia/Tokyo", Message: "Olá"},
	}, announced: map[uint64]bool{3: true}}
	service := newService(repository)
	announcements, err := service.Due(time.Date(2026, 1, 1, 15, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if len(announcements) != 1 || announcements[0].UserID != 1 || announcements[0].Age != 26 || announcements[0].LocalDate.Day() != 2 {
		t.Fatalf("Due() = %#v", announcements)
	}
}

func TestNextUsesLocalDatesAndSkipsTodaysBirthday(t *testing.T) {
	repository := &fakeRepository{birthdays: []*entity.Birthday{
		{UserID: 1, Birthday: time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC), TimeZone: "Asia/Tokyo"},
		{UserID: 2, Birthday: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), TimeZone: "UTC"},
		{UserID: 3, Birthday: time.Date(2000, 1, 3, 0, 0, 0, 0, time.UTC), TimeZone: "UTC"},
	}}
	next, err := newService(repository).Next(time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))
	if err != nil || next == nil || next.UserID != 1 || !next.OccursAt.Equal(time.Date(2026, 1, 2, 0, 0, 0, 0, time.FixedZone("JST", 9*60*60))) {
		t.Fatalf("Next() = %#v, %v", next, err)
	}
}

func TestNextUsesFebruaryTwentyEighthForLeapDayBirthday(t *testing.T) {
	repository := &fakeRepository{birthdays: []*entity.Birthday{{
		UserID: 1, Birthday: time.Date(2000, 2, 29, 0, 0, 0, 0, time.UTC), TimeZone: "UTC",
	}}}
	next, err := newService(repository).Next(time.Date(2025, 2, 27, 12, 0, 0, 0, time.UTC))
	if err != nil || next == nil || !next.OccursAt.Equal(time.Date(2025, 2, 28, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("Next() = %#v, %v", next, err)
	}
}

func TestNextReturnsNoBirthdayAndRepositoryErrors(t *testing.T) {
	if next, err := newService(&fakeRepository{}).Next(time.Now()); err != nil || next != nil {
		t.Fatalf("Next() = %#v, %v", next, err)
	}
	want := errors.New("list")
	if _, err := newService(&fakeRepository{listErr: want}).Next(time.Now()); !errors.Is(err, want) {
		t.Fatalf("Next() error = %v", err)
	}
}

func TestDueCelebratesLeapBirthdayOnFebruaryTwentyEighth(t *testing.T) {
	repository := &fakeRepository{birthdays: []*entity.Birthday{{
		UserID: 1, Birthday: time.Date(2000, 2, 29, 0, 0, 0, 0, time.UTC),
		TimeZone: "UTC", Message: "Olá",
	}}}
	announcements, err := newService(repository).Due(time.Date(2025, 2, 28, 12, 0, 0, 0, time.UTC))
	if err != nil || len(announcements) != 1 || announcements[0].Age != 25 {
		t.Fatalf("Due() = %#v, %v", announcements, err)
	}
	announcements, err = newService(repository).Due(time.Date(2024, 2, 28, 12, 0, 0, 0, time.UTC))
	if err != nil || len(announcements) != 0 {
		t.Fatalf("leap-year Due() = %#v, %v", announcements, err)
	}
}

func TestDueUsesConfiguredDefaultMessageForEmptyStoredMessage(t *testing.T) {
	defaultMessage := &fakeDefaultMessageProvider{message: "Feliz aniversário, {mention}! Hoje {name} completa {age} anos."}
	repository := &fakeRepository{birthdays: []*entity.Birthday{{
		UserID: 1, Name: "Leo", Birthday: time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC), TimeZone: "UTC",
	}}}
	announcements, err := NewService(repository, defaultMessage).Due(time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC))
	if err != nil || len(announcements) != 1 || announcements[0].Message != defaultMessage.message || defaultMessage.calls != 1 {
		t.Fatalf("Due() = %#v, %v; default calls = %d", announcements, err, defaultMessage.calls)
	}
}

func TestDueDoesNotReadConfiguredDefaultForStoredMessage(t *testing.T) {
	defaultMessage := &fakeDefaultMessageProvider{message: "default"}
	repository := &fakeRepository{birthdays: []*entity.Birthday{{
		UserID: 1, Name: "Leo", Birthday: time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC), TimeZone: "UTC", Message: "Personalizada",
	}}}
	announcements, err := NewService(repository, defaultMessage).Due(time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC))
	if err != nil || len(announcements) != 1 || announcements[0].Message != "Personalizada" || defaultMessage.calls != 0 {
		t.Fatalf("Due() = %#v, %v; default calls = %d", announcements, err, defaultMessage.calls)
	}
}

func TestDueReturnsDefaultMessageError(t *testing.T) {
	want := errors.New("default message unavailable")
	repository := &fakeRepository{birthdays: []*entity.Birthday{{
		UserID: 1, Birthday: time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC), TimeZone: "UTC",
	}}}
	if _, err := NewService(repository, &fakeDefaultMessageProvider{err: want}).Due(time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC)); !errors.Is(err, want) {
		t.Fatalf("Due() error = %v", err)
	}
}

func TestDueErrors(t *testing.T) {
	want := errors.New("failure")
	if _, err := newService(&fakeRepository{listErr: want}).Due(time.Now()); !errors.Is(err, want) {
		t.Fatalf("list error = %v", err)
	}
	invalidZone := &fakeRepository{birthdays: []*entity.Birthday{{UserID: 1, TimeZone: "bad"}}}
	if _, err := newService(invalidZone).Due(time.Now()); err == nil {
		t.Fatal("invalid stored time zone accepted")
	}
	ledgerFailure := &fakeRepository{
		birthdays:       []*entity.Birthday{{UserID: 1, Birthday: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), TimeZone: "UTC"}},
		announcementErr: want,
	}
	if _, err := newService(ledgerFailure).Due(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)); !errors.Is(err, want) {
		t.Fatalf("ledger error = %v", err)
	}
}

func newService(repository Repository) *Service {
	return NewService(repository, &fakeDefaultMessageProvider{})
}

func TestMarkAnnounced(t *testing.T) {
	repository := &fakeRepository{}
	value := time.Date(2026, 1, 2, 13, 0, 0, 0, time.FixedZone("test", 3600))
	if err := newService(repository).MarkAnnounced(7, value); err != nil {
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
	found           *entity.Birthday
	findErr         error
	updateFound     *bool
	updateErr       error
	updatedValue    any
}

type fakeDefaultMessageProvider struct {
	message string
	calls   int
	err     error
}

func (p *fakeDefaultMessageProvider) BirthdayDefaultMessage() (string, error) {
	p.calls++
	return p.message, p.err
}

func (r *fakeRepository) ListByMonth(month time.Month) ([]*entity.Birthday, error) {
	r.month = month
	return r.monthBirthdays, r.listErr
}

func (r *fakeRepository) List() ([]*entity.Birthday, error) {
	return r.birthdays, r.listErr
}

func (r *fakeRepository) FindByUserID(uint64) (*entity.Birthday, error) {
	return r.found, r.findErr
}

func (r *fakeRepository) Save(birthday *entity.Birthday) error {
	r.saved = birthday
	return r.saveErr
}

func (r *fakeRepository) UpdateName(_ uint64, value string) (bool, error) {
	return r.recordUpdate(value)
}

func (r *fakeRepository) UpdateBirthday(_ uint64, value time.Time) (bool, error) {
	return r.recordUpdate(value)
}

func (r *fakeRepository) UpdateTimeZone(_ uint64, value string) (bool, error) {
	return r.recordUpdate(value)
}

func (r *fakeRepository) UpdateMessage(_ uint64, value string) (bool, error) {
	return r.recordUpdate(value)
}

func (r *fakeRepository) recordUpdate(value any) (bool, error) {
	r.updatedValue = value
	if r.updateErr != nil {
		return false, r.updateErr
	}
	return r.updateFound == nil || *r.updateFound, nil
}

func (r *fakeRepository) WasAnnounced(userID uint64, _ time.Time) (bool, error) {
	return r.announced[userID], r.announcementErr
}

func (r *fakeRepository) MarkAnnounced(userID uint64, localDate time.Time) error {
	r.markedUser, r.markedDate = userID, localDate
	return r.announcementErr
}
