package pages

import (
	"log/slog"
	"net/http"
	"net/mail"
	"strings"

	"github.com/alexpls/untils/internal/db"
	"github.com/alexpls/untils/internal/docs"
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

// DocsHome handles GET /docs
func (h *Handlers) DocsHome(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, docs.CurrentSite().IndexPath, http.StatusFound)
}

// DocsPage handles GET /docs/{doc_path...}
func (h *Handlers) DocsPage(w http.ResponseWriter, r *http.Request) {
	docPath := docs.NormalizePath(strings.TrimPrefix(r.URL.Path, "/"))
	page, ok := docs.CurrentSite().Page(docPath)
	if !ok {
		http.NotFound(w, r)
		return
	}

	h.renderDocsPage(w, r, page)
}

func (h *Handlers) renderDocsPage(w http.ResponseWriter, r *http.Request, page docs.Page) {
	if err := docs.PageView(page, docs.CurrentSite().NavSections).Render(r.Context(), w); err != nil {
		h.logger.ErrorContext(r.Context(), "error rendering docs page", "error", err, "path", page.Path)
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
