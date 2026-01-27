package monitor

import (
	"errors"
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
)

// DefaultCheckSchedule is the default cron schedule for monitor checks (every 6 hours).
const DefaultCheckSchedule = "0 */6 * * *"

// Schedule frequency bounds
const (
	MinScheduleInterval = 1 * time.Hour
	MaxScheduleInterval = 7 * 24 * time.Hour // 1 week
)

var (
	ErrScheduleTooFrequent   = errors.New("schedule is too frequent (minimum interval is 1 hour)")
	ErrScheduleTooInfrequent = errors.New("schedule is too infrequent (maximum interval is 1 week)")
)

// cronParser parses standard 5-field cron expressions (minute, hour, day-of-month, month, day-of-week).
var cronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

// parseSchedule parses a cron expression and returns a Schedule.
func parseSchedule(expr string) (cron.Schedule, error) {
	schedule, err := cronParser.Parse(expr)
	if err != nil {
		return nil, fmt.Errorf("invalid cron expression %q: %w", expr, err)
	}
	return schedule, nil
}

// nextCheckTime calculates the next check time based on a cron schedule expression.
// The cron expression is interpreted in the timezone of the 'from' time.
// The returned time preserves the timezone of 'from'.
func nextCheckTime(schedule string, from time.Time) (time.Time, error) {
	sched, err := parseSchedule(schedule)
	if err != nil {
		return time.Time{}, err
	}
	return sched.Next(from), nil
}

// validateSchedule validates a cron schedule expression and checks that it falls
// within the allowed frequency bounds (between 1 hour and 1 week).
func validateSchedule(expr string) error {
	sched, err := parseSchedule(expr)
	if err != nil {
		return err
	}

	// Calculate the interval by finding two consecutive runs
	now := time.Now()
	first := sched.Next(now)
	second := sched.Next(first)
	interval := second.Sub(first)

	if interval < MinScheduleInterval {
		return ErrScheduleTooFrequent
	}
	if interval > MaxScheduleInterval {
		return ErrScheduleTooInfrequent
	}

	return nil
}
