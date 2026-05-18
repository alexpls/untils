package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/alexpls/untils/internal/logging"
	"github.com/alexpls/untils/internal/reqcontext"
)

func (h *Handlers) RequireToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key, ok := bearerToken(r.Header.Get("Authorization"))
		if !ok {
			writeUnauthorized(w)
			return
		}

		token, err := h.service.AuthenticateToken(r.Context(), key)
		if err != nil {
			if errors.Is(err, ErrInvalidToken) {
				writeUnauthorized(w)
				return
			}

			h.logger.ErrorContext(r.Context(), "authenticating api token", "error", err)
			writeError(w, http.StatusInternalServerError, "internal_error", "An internal error occurred.")
			return
		}

		logging.GetOrCreateFromContext(r.Context(), func() *TokenLogEvent {
			return &TokenLogEvent{
				ID:   token.ID,
				Name: token.Name,
			}
		})

		next.ServeHTTP(w, r.WithContext(reqcontext.ContextWithAPIToken(r.Context(), token)))
	})
}

func bearerToken(header string) (string, bool) {
	scheme, token, ok := strings.Cut(header, " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") {
		return "", false
	}
	// RFC 7235 says the auth scheme is case-insensitive. Trim whitespace
	// around the token so clients that accidentally include padding still
	// authenticate cleanly.
	token = strings.TrimSpace(token)
	if token == "" {
		return "", false
	}
	return token, true
}
