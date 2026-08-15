// Package job provides Padinho's small in-process recurring-job scheduler.
package job

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"
)

var ErrInvalidSchedule = errors.New("job interval and task are required")

type scheduled struct {
	interval time.Duration
	run      func() error
}

// Scheduler executes independent recurring tasks without overlapping a task
// with itself. It deliberately uses the standard library rather than a cron dependency.
type Scheduler struct {
	logger *slog.Logger
	jobs   []scheduled
	wait   sync.WaitGroup
}

func NewScheduler(logger *slog.Logger) *Scheduler {
	return &Scheduler{logger: logger}
}

// Every registers a task to run once per interval.
func (s *Scheduler) Every(interval time.Duration, task func() error) error {
	if interval <= 0 || task == nil {
		return ErrInvalidSchedule
	}
	s.jobs = append(s.jobs, scheduled{interval: interval, run: task})
	return nil
}

// Start launches all registered jobs. The first execution occurs after one interval.
func (s *Scheduler) Start(ctx context.Context) {
	for _, current := range s.jobs {
		current := current
		s.wait.Add(1)
		go func() {
			defer s.wait.Done()
			ticker := time.NewTicker(current.interval)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					if err := current.run(); err != nil {
						s.logger.Error("scheduled job failed", "error", err)
					}
				}
			}
		}()
	}
}

// Wait blocks until cancellation has stopped every scheduler goroutine.
func (s *Scheduler) Wait() {
	s.wait.Wait()
}
