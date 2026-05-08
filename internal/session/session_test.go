package session

import (
	"context"
	"log/slog"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSessionData_SetAndPopFlash(t *testing.T) {
	data := SessionData{}

	data.SetFlash(FlashTypeAlert, "Password changed.")
	assert.Equal(t, "Password changed.", data.PopFlash(FlashTypeAlert))
	assert.Equal(t, "", data.PopFlash(FlashTypeAlert))
}

func TestManagerNewUsesSecureCookieConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		secureCookies bool
	}{
		{name: "secure cookies enabled", secureCookies: true},
		{name: "secure cookies disabled", secureCookies: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			manager := NewManager(nil, nil, tt.secureCookies, slog.Default())
			recorder := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/", nil)
			req = req.WithContext(context.WithValue(req.Context(), sessionCtxKey, &Session{}))

			manager.New(req, recorder)

			cookies := recorder.Result().Cookies()
			if len(cookies) != 1 {
				t.Fatalf("got %d cookies, want 1", len(cookies))
			}
			assert.Equal(t, tt.secureCookies, cookies[0].Secure)
		})
	}
}
