package monitor

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
)

// DefaultCheckSchedule is the default cron schedule for monitor checks (8am, 12pm, 4pm, 8pm every day).
const DefaultCheckSchedule = "0 8,12,16,20 * * *"

// Schedule frequency bounds
const (
	MinScheduleInterval = 1 * time.Hour
	MaxScheduleInterval = 7 * 24 * time.Hour // 1 week
)

var (
	ErrScheduleTooFrequent         = errors.New("schedule is too frequent (minimum interval is 1 hour)")
	ErrScheduleTooInfrequent       = errors.New("schedule is too infrequent (maximum interval is 1 week)")
	ErrScheduleNeedsHourAndWeekday = errors.New("schedule needs at least one hour and one weekday")
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

	// has both an hour and a day of week been selected?
	specSched, ok := sched.(*cron.SpecSchedule)
	if !ok {
		panic("schedule is not a SpecSchedule")
	}
	hours := bitsToSlice(specSched.Hour, 0, 23)
	weekdays := bitsToSlice(specSched.Dow, 0, 6)

	if len(hours) == 0 || len(weekdays) == 0 {
		return ErrScheduleNeedsHourAndWeekday
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

func formatCronExpression(expression string) string {
	if expression == "" {
		return ""
	}

	sched, err := parseSchedule(expression)
	if err != nil {
		return ""
	}

	specSched, ok := sched.(*cron.SpecSchedule)
	if !ok {
		panic("schedule is not a SpecSchedule")
	}

	hours := bitsToSlice(specSched.Hour, 0, 23)
	weekdays := bitsToSlice(specSched.Dow, 0, 6)

	var timeStr string
	if len(hours) == 24 {
		timeStr = "every hour"
	} else if len(hours) == 1 {
		timeStr = formatHour(hours[0])
	} else {
		hourStrs := make([]string, len(hours))
		for i, h := range hours {
			hourStrs[i] = formatHour(h)
		}
		if len(hourStrs) == 2 {
			timeStr = hourStrs[0] + " and " + hourStrs[1]
		} else {
			timeStr = strings.Join(hourStrs[:len(hourStrs)-1], ", ") + ", and " + hourStrs[len(hourStrs)-1]
		}
	}

	var dayStr string
	if len(weekdays) == 7 {
		dayStr = "every day"
	} else if len(weekdays) == 1 {
		dayStr = formatWeekday(weekdays[0]) + "s"
	} else {
		dayNames := make([]string, len(weekdays))
		for i, d := range weekdays {
			dayNames[i] = formatWeekday(d)
		}
		if len(dayNames) == 2 {
			dayStr = dayNames[0] + " and " + dayNames[1]
		} else {
			dayStr = strings.Join(dayNames[:len(dayNames)-1], ", ") + ", and " + dayNames[len(dayNames)-1]
		}
	}

	if len(weekdays) == 7 {
		if len(hours) == 24 {
			return "Every hour, every day"
		}
		return timeStr + " every day"
	}

	if len(hours) == 24 {
		return "Every hour on " + dayStr
	}

	return timeStr + " on " + dayStr
}

// bitsToSlice extracts set bits from a uint64 bit field into a sorted slice of integers.
func bitsToSlice(bits uint64, min, max int) []int {
	var result []int
	for i := min; i <= max; i++ {
		if bits&(1<<uint(i)) != 0 {
			result = append(result, i)
		}
	}
	return result
}

// formatHour formats an hour (0-23) as a friendly time string like "9am" or "12pm".
func formatHour(hour int) string {
	switch {
	case hour == 0:
		return "12am"
	case hour < 12:
		return fmt.Sprintf("%dam", hour)
	case hour == 12:
		return "12pm"
	default:
		return fmt.Sprintf("%dpm", hour-12)
	}
}

// formatWeekday formats a weekday number (0=Sunday) as the day name.
func formatWeekday(day int) string {
	days := []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}
	if day >= 0 && day < len(days) {
		return days[day]
	}
	return fmt.Sprintf("day%d", day)
}
