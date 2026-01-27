package monitor

import (
	"errors"
	"testing"
	"time"

	"github.com/alexpls/untils/internal/must"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSchedule(t *testing.T) {
	tests := []struct {
		name    string
		expr    string
		wantErr bool
	}{
		{"every 6 hours", "0 */6 * * *", false},
		{"every friday at 9am", "0 9 * * 5", false},
		{"daily at 9am and 5pm", "0 9,17 * * *", false},
		{"weekdays at 9am", "0 9 * * 1-5", false},
		{"invalid expression", "invalid", true},
		{"empty expression", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseSchedule(tt.expr)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateSchedule(t *testing.T) {
	tests := []struct {
		name    string
		expr    string
		wantErr error
	}{
		// Valid schedules
		{"every hour", "0 * * * *", nil},
		{"every 6 hours", "0 */6 * * *", nil},
		{"daily at 9am", "0 9 * * *", nil},
		{"daily at 9am and 5pm", "0 9,17 * * *", nil},
		{"every friday at 9am", "0 9 * * 5", nil},
		{"default schedule", DefaultCheckSchedule, nil},

		// Too frequent
		{"every minute", "* * * * *", ErrScheduleTooFrequent},
		{"every 30 minutes", "*/30 * * * *", ErrScheduleTooFrequent},

		// Too infrequent
		{"every 2 weeks (monthly on 1st and 15th)", "0 9 1,15 * *", ErrScheduleTooInfrequent},

		// Invalid syntax
		{"invalid expression", "invalid", nil}, // parseSchedule error, not frequency error
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSchedule(tt.expr)
			if tt.wantErr != nil {
				assert.True(t, errors.Is(err, tt.wantErr), "expected %v, got %v", tt.wantErr, err)
			} else if tt.expr == "invalid" {
				assert.Error(t, err, "expected parse error for invalid expression")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNextCheckTime(t *testing.T) {
	t.Run("past", func(t *testing.T) {
		schedule := "0 9 * * *" // daily at 9am
		yesterday := time.Now().Add(-24 * time.Hour)

		next, err := nextCheckTime(schedule, yesterday)
		require.NoError(t, err)
		assert.Equal(t, 9, next.Hour())
		assert.Equal(t, 0, next.Minute())
	})

	t.Run("preserves timezone", func(t *testing.T) {
		schedule := "0 9 * * *"
		now := time.Now().In(must.NoErrVal((time.LoadLocation("America/New_York"))))

		next, err := nextCheckTime(schedule, now)
		require.NoError(t, err)
		assert.Equal(t, 9, next.Hour())
		assert.Equal(t, 0, next.Minute())
		assert.Equal(t, "America/New_York", next.Location().String())
	})
}
