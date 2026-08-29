package job

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/robfig/cron/v3"
)

func TestSchedulerRunsAndStopsJobs(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	scheduler := newScheduler(logger, cron.WithSeconds())
	var calls atomic.Int32
	if err := scheduler.Schedule("* * * * * *", func() error {
		calls.Add(1)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	scheduler.Start(ctx)
	deadline := time.After(time.Second)
	for calls.Load() == 0 {
		select {
		case <-deadline:
			t.Fatal("scheduled task did not run")
		default:
			time.Sleep(time.Millisecond)
		}
	}
	cancel()
	scheduler.Wait()
}

func TestNewSchedulerUsesUTCAndHandlesNilLogger(t *testing.T) {
	scheduler := NewScheduler(nil)
	if scheduler.logger == nil {
		t.Fatal("scheduler logger is nil")
	}
	if scheduler.cron.Location() != time.UTC {
		t.Fatalf("scheduler location = %v, want UTC", scheduler.cron.Location())
	}
}

func TestSchedulerLogsTaskErrorsAndValidatesRegistration(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	scheduler := newScheduler(logger, cron.WithSeconds())
	if err := scheduler.Schedule("", func() error { return nil }); !errors.Is(err, ErrInvalidSchedule) {
		t.Fatalf("empty expression error = %v", err)
	}
	if err := scheduler.Schedule("* * * * *", nil); !errors.Is(err, ErrInvalidSchedule) {
		t.Fatalf("nil task error = %v", err)
	}
	if err := scheduler.Schedule("not a cron expression", func() error { return nil }); err == nil {
		t.Fatal("invalid expression was accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := scheduler.Schedule("* * * * * *", func() error { cancel(); return ErrInvalidSchedule }); err != nil {
		t.Fatal(err)
	}
	scheduler.Start(ctx)
	scheduler.Wait()
	if !bytes.Contains(logs.Bytes(), []byte("scheduled job failed")) {
		t.Fatalf("error log = %q", logs.String())
	}
}

func TestSchedulerSkipsOverlappingRuns(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	scheduler := newScheduler(logger, cron.WithSeconds())
	started := make(chan struct{}, 1)
	release := make(chan struct{})
	var active atomic.Int32
	var maximumActive atomic.Int32
	if err := scheduler.Schedule("* * * * * *", func() error {
		current := active.Add(1)
		for {
			maximum := maximumActive.Load()
			if current <= maximum || maximumActive.CompareAndSwap(maximum, current) {
				break
			}
		}
		select {
		case started <- struct{}{}:
		default:
		}
		<-release
		active.Add(-1)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	scheduler.Start(ctx)
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("scheduled task did not start")
	}
	time.Sleep(1500 * time.Millisecond)
	close(release)
	cancel()
	scheduler.Wait()
	if got := maximumActive.Load(); got != 1 {
		t.Fatalf("maximum concurrent runs = %d, want 1", got)
	}
}

func TestSlogCronLoggerLogsErrors(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	slogCronLogger{logger: logger}.Error(errors.New("failure"), "job failed", "entry", 1)
	if !bytes.Contains(logs.Bytes(), []byte("cron job failed")) || !bytes.Contains(logs.Bytes(), []byte("error=failure")) {
		t.Fatalf("error log = %q", logs.String())
	}
}
