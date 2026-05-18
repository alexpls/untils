package pages

import (
	"log/slog"
	"net/http"
	"net/mail"

	"github.com/alexpls/untils/internal/db"
	"github.com/alexpls/untils/internal/models"
	"github.com/starfederation/datastar-go/datastar"
)

// Handlers contains HTTP handlers for public pages
type Handlers struct {
	queries *models.Queries
	db      db.DB
	logger  *slog.Logger
}

// NewHandlers creates a new Handlers instance
func NewHandlers(queries *models.Queries, db db.DB, logger *slog.Logger) *Handlers {
	return &Handlers{
		queries: queries,
		db:      db,
		logger:  logger,
	}
}

// Home handles GET /
func (h *Handlers) Home(w http.ResponseWriter, r *http.Request) {
	if err := HomePage().Render(r.Context(), w); err != nil {
		h.logger.ErrorContext(r.Context(), "error rendering home page", "error", err)
	}
}

// SubscribeEmail handles POST /subscribe
func (h *Handlers) SubscribeEmail(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.logger.ErrorContext(r.Context(), "error parsing subscribe form", "error", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	email := r.FormValue("email")

	if _, err := mail.ParseAddress(email); err != nil {
		sse := datastar.NewSSE(w, r)
		if err := sse.PatchElementTempl(emailFormInner("Please enter a valid email address.")); err != nil {
			h.logger.ErrorContext(r.Context(), "error patching element", "error", err)
		}
		return
	}

	if err := h.queries.CreateEmailSubscriber(r.Context(), h.db, email); err != nil {
		h.logger.ErrorContext(r.Context(), "error creating email subscriber", "error", err)
		sse := datastar.NewSSE(w, r)
		if err := sse.PatchElementTempl(emailFormInner("Something went wrong. Please try again.")); err != nil {
			h.logger.ErrorContext(r.Context(), "error patching element", "error", err)
		}
		return
	}

	sse := datastar.NewSSE(w, r)
	if err := sse.PatchElementTempl(emailFormSuccess()); err != nil {
		h.logger.ErrorContext(r.Context(), "error patching element", "error", err)
	}
}
