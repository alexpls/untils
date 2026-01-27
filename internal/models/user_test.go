package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestUserNow(t *testing.T) {
	tests := []struct {
		name     string
		timezone string
		wantTz   string
	}{
		{"valid timezone", "America/New_York", "America/New_York"},
		{"UTC", "UTC", "UTC"},
		{"invalid timezone falls back to UTC", "Invalid/Timezone", "UTC"},
		{"empty timezone falls back to UTC", "", "UTC"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &User{Timezone: tt.timezone}
			now := user.Now()

			wantLoc, _ := time.LoadLocation(tt.wantTz)
			assert.Equal(t, wantLoc.String(), now.Location().String())
			assert.WithinDuration(t, time.Now(), now, time.Second)
		})
	}
}
