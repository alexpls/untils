package pages

import (
	"net/http"
)

// Handlers contains HTTP handlers for public pages
type Handlers struct{}

// NewHandlers creates a new Handlers instance
func NewHandlers() *Handlers {
	return &Handlers{}
}

// Home handles GET /
func (h *Handlers) Home(w http.ResponseWriter, r *http.Request) {
	HomePage().Render(r.Context(), w)
}
