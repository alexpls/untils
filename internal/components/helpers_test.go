package components

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTimeAgoTextAt(t *testing.T) {
	now := time.Date(2026, time.May, 6, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		at   time.Time
		want string
	}{
		{
			name: "just now",
			at:   now,
			want: "just now",
		},
		{
			name: "past second",
			at:   now.Add(-time.Second),
			want: "1 second ago",
		},
		{
			name: "past plural seconds",
			at:   now.Add(-2 * time.Second),
			want: "2 seconds ago",
		},
		{
			name: "rounds to nearest minute",
			at:   now.Add(-90 * time.Second),
			want: "2 minutes ago",
		},
		{
			name: "past hour",
			at:   now.Add(-time.Hour),
			want: "1 hour ago",
		},
		{
			name: "future day",
			at:   now.Add(24 * time.Hour),
			want: "in 1 day",
		},
		{
			name: "future plural months",
			at:   now.Add(60 * 24 * time.Hour),
			want: "in 2 months",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, TimeAgoTextAt(now, tt.at))
		})
	}
}
