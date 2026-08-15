package birthday

import (
	"errors"
	"strings"
	"testing"
	"time"

	appbirthday "github.com/kiLeo13/SaladaFun/discord/padinho/internal/application/birthday"
)

func TestJobSendsAndMarksEveryAnnouncement(t *testing.T) {
	localDate := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	service := &fakeService{due: []appbirthday.Announcement{
		{UserID: 1, LocalDate: localDate},
		{UserID: 2, LocalDate: localDate},
	}}
	sender := &fakeSender{}
	job := New(service, sender)
	job.now = func() time.Time { return localDate }
	if err := job.Run(); err != nil {
		t.Fatal(err)
	}
	if len(sender.sent) != 2 || len(service.marked) != 2 {
		t.Fatalf("sent = %#v, marked = %#v", sender.sent, service.marked)
	}
}

func TestJobReturnsDueError(t *testing.T) {
	want := errors.New("due")
	err := New(&fakeService{err: want}, &fakeSender{}).Run()
	if !errors.Is(err, want) {
		t.Fatalf("Run() error = %v", err)
	}
}

func TestJobContinuesAfterDeliveryAndLedgerFailures(t *testing.T) {
	service := &fakeService{
		due:     []appbirthday.Announcement{{UserID: 1}, {UserID: 2}, {UserID: 3}},
		markErr: map[uint64]error{2: errors.New("mark")},
	}
	sender := &fakeSender{errors: map[uint64]error{1: errors.New("send")}}
	err := New(service, sender).Run()
	if err == nil || !strings.Contains(err.Error(), "user 1") || !strings.Contains(err.Error(), "user 2") {
		t.Fatalf("Run() error = %v", err)
	}
	if len(sender.sent) != 3 || len(service.marked) != 2 {
		t.Fatalf("sent = %#v, marked = %#v", sender.sent, service.marked)
	}
}

type fakeService struct {
	due     []appbirthday.Announcement
	marked  []uint64
	err     error
	markErr map[uint64]error
}

func (s *fakeService) Due(time.Time) ([]appbirthday.Announcement, error) {
	return s.due, s.err
}

func (s *fakeService) MarkAnnounced(userID uint64, _ time.Time) error {
	s.marked = append(s.marked, userID)
	return s.markErr[userID]
}

type fakeSender struct {
	sent   []uint64
	errors map[uint64]error
}

func (s *fakeSender) Send(announcement appbirthday.Announcement) error {
	s.sent = append(s.sent, announcement.UserID)
	return s.errors[announcement.UserID]
}
