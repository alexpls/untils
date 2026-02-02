package monitor

import (
	"errors"
	"testing"
	"time"

	"github.com/alexpls/untils/internal/must"
	"github.com/stretchr/testify/assert"
)

func TestValidateFrequency(t *testing.T) {
	tests := []struct {
		name    string
		minutes int32
		wantErr error
	}{
		// Valid frequencies
		{"every hour", 60, nil},
		{"every 8 hours", 480, nil},
		{"every day", 1440, nil},
		{"every 2 days", 2880, nil},
		{"every week", 10080, nil},
		{"default", DefaultCheckFrequencyMinutes, nil},

		// Too frequent
		{"every 30 minutes", 30, ErrFrequencyTooFrequent},
		{"every minute", 1, ErrFrequencyTooFrequent},

		// Too infrequent
		{"every 2 weeks", 20160, ErrFrequencyTooInfrequent},
		{"every month", 43200, ErrFrequencyTooInfrequent},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFrequency(tt.minutes)
			if tt.wantErr != nil {
				assert.True(t, errors.Is(err, tt.wantErr), "expected %v, got %v", tt.wantErr, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNextCheckTime(t *testing.T) {
	t.Run("adds frequency to from time", func(t *testing.T) {
		frequency := int32(1440) // daily
		now := time.Now()

		next := nextCheckTime(frequency, now)
		expected := now.Add(24 * time.Hour)

		// Allow small time difference due to test execution
		diff := next.Sub(expected)
		assert.Less(t, diff.Abs(), time.Second)
	})

	t.Run("preserves timezone", func(t *testing.T) {
		frequency := int32(60) // hourly
		now := time.Now().In(must.NoErrVal((time.LoadLocation("America/New_York"))))

		next := nextCheckTime(frequency, now)
		assert.Equal(t, "America/New_York", next.Location().String())
	})

	t.Run("different frequencies", func(t *testing.T) {
		baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

		tests := []struct {
			frequency int32
			expected  time.Time
		}{
			{60, time.Date(2024, 1, 1, 13, 0, 0, 0, time.UTC)},   // +1 hour
			{480, time.Date(2024, 1, 1, 20, 0, 0, 0, time.UTC)},  // +8 hours
			{1440, time.Date(2024, 1, 2, 12, 0, 0, 0, time.UTC)}, // +1 day
			{2880, time.Date(2024, 1, 3, 12, 0, 0, 0, time.UTC)}, // +2 days
		}

		for _, tt := range tests {
			next := nextCheckTime(tt.frequency, baseTime)
			assert.Equal(t, tt.expected, next)
		}
	})
}
