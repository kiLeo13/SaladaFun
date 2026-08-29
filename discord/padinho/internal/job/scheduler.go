// Package job provides Padinho's cron-backed in-process job scheduler.
package job

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

var ErrInvalidSchedule = errors.New("job schedule and task are required")

// Scheduler executes cron-scheduled tasks without overlapping the same task.
type Scheduler struct {
	logger *slog.Logger
	cron   *cron.Cron
	wait   sync.WaitGroup
}

// NewScheduler constructs a UTC scheduler for wall-clock cron expressions.
func NewScheduler(logger *slog.Logger) *Scheduler {
	return newScheduler(logger)
}

func newScheduler(logger *slog.Logger, options ...cron.Option) *Scheduler {
	if logger == nil {
		logger = slog.Default()
	}
	cronLogger := slogCronLogger{logger: logger}
	defaults := []cron.Option{
		cron.WithLocation(time.UTC),
		cron.WithLogger(cronLogger),
		cron.WithChain(
			cron.SkipIfStillRunning(cronLogger),
			cron.Recover(cronLogger),
		),
	}
	return &Scheduler{
		logger: logger,
		cron:   cron.New(append(defaults, options...)...),
	}
}

// Schedule registers a task for a standard cron expression.
func (s *Scheduler) Schedule(expression string, task func() error) error {
	if expression == "" || task == nil {
		return ErrInvalidSchedule
	}
	if _, err := s.cron.AddFunc(expression, func() {
		if err := task(); err != nil {
			s.logger.Error("scheduled job failed", "error", err)
		}
	}); err != nil {
		return fmt.Errorf("register cron schedule %q: %w", expression, err)
	}
	return nil
}

// Start launches all registered jobs and stops them after context cancellation.
func (s *Scheduler) Start(ctx context.Context) {
	s.cron.Start()
	s.wait.Go(func() {
		<-ctx.Done()
		<-s.cron.Stop().Done()
	})
}

// Wait blocks until cancellation stops the scheduler and its running jobs.
func (s *Scheduler) Wait() {
	s.wait.Wait()
}

type slogCronLogger struct {
	logger *slog.Logger
}

// Info records cron lifecycle events at debug level.
func (l slogCronLogger) Info(message string, values ...any) {
	l.logger.Debug("cron "+message, values...)
}

// Error records cron failures through Padinho's structured logger.
func (l slogCronLogger) Error(err error, message string, values ...any) {
	values = append(values, "error", err)
	l.logger.Error("cron "+message, values...)
}
