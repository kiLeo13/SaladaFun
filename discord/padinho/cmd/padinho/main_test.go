package main

import (
	"testing"
	"time"

	"github.com/robfig/cron/v3"
)

// TestBirthdayCheckScheduleRunsAtHourBoundary prevents startup-relative checks.
func TestBirthdayCheckScheduleRunsAtHourBoundary(t *testing.T) {
	schedule, err := cron.ParseStandard(birthdayCheckSchedule)
	if err != nil {
		t.Fatalf("parse birthday schedule: %v", err)
	}
	reference := time.Date(2026, time.January, 1, 23, 59, 45, 0, time.UTC)
	want := time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC)
	if got := schedule.Next(reference); !got.Equal(want) {
		t.Fatalf("next birthday check = %v, want %v", got, want)
	}
}
