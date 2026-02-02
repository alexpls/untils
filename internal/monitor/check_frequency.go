package monitor

import (
	"errors"
	"time"
)

// DefaultCheckFrequencyMinutes is the default check frequency (24 hours = 1440 minutes).
const DefaultCheckFrequencyMinutes = 1440

// Frequency bounds in minutes
const (
	MinCheckFrequencyMinutes = 60    // 1 hour
	MaxCheckFrequencyMinutes = 10080 // 1 week (7 * 24 * 60)
)

var (
	ErrFrequencyTooFrequent   = errors.New("check frequency is too frequent (minimum is 1 hour)")
	ErrFrequencyTooInfrequent = errors.New("check frequency is too infrequent (maximum is 1 week)")
)

var FrequencyOptions = []struct {
	Minutes int32
	Label   string
}{
	{60, "every hour"},
	{480, "every 8 hours"},
	{1440, "every day"},
	{2880, "every 2 days"},
	{10080, "every week"},
}

// nextCheckTime calculates the next check time based on frequency and from time.
func nextCheckTime(frequencyMinutes int32, from time.Time) time.Time {
	return from.Add(time.Duration(frequencyMinutes) * time.Minute)
}

// validateFrequency validates that the frequency is within allowed bounds.
func validateFrequency(minutes int32) error {
	if minutes < MinCheckFrequencyMinutes {
		return ErrFrequencyTooFrequent
	}
	if minutes > MaxCheckFrequencyMinutes {
		return ErrFrequencyTooInfrequent
	}
	return nil
}
