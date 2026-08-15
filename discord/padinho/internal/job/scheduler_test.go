package job

import (
	"context"
	"io"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"
)

func TestSchedulerRunsAndStopsJobs(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	scheduler := NewScheduler(logger)
	var calls atomic.Int32
	if err := scheduler.Every(time.Millisecond, func() error {
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

func TestSchedulerLogsTaskErrorsAndValidatesRegistration(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	scheduler := NewScheduler(logger)
	if err := scheduler.Every(0, func() error { return nil }); err != ErrInvalidSchedule {
		t.Fatalf("zero interval error = %v", err)
	}
	if err := scheduler.Every(time.Minute, nil); err != ErrInvalidSchedule {
		t.Fatalf("nil task error = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	if err := scheduler.Every(time.Millisecond, func() error { cancel(); return ErrInvalidSchedule }); err != nil {
		t.Fatal(err)
	}
	scheduler.Start(ctx)
	scheduler.Wait()
}
