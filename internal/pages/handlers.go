package pages

import (
	"log/slog"
	"net/http"
)

// Handlers contains HTTP handlers for public pages
type Handlers struct {
	logger *slog.Logger
}

// NewHandlers creates a new Handlers instance
func NewHandlers(logger *slog.Logger) *Handlers {
	return &Handlers{
		logger: logger,
	}
}

// Home handles GET /
func (h *Handlers) Home(w http.ResponseWriter, r *http.Request) {
	if err := HomePage().Render(r.Context(), w); err != nil {
		h.logger.Error("error rendering home page", "error", err)
	}
}
