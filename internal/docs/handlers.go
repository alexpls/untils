package docs

import (
	"log/slog"
	"net/http"
	"strings"
)

type Handlers struct {
	logger *slog.Logger
}

func NewHandlers(logger *slog.Logger) *Handlers {
	return &Handlers{logger: logger}
}

// Home handles GET /docs.
func (h *Handlers) Home(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, currentSite.IndexPath, http.StatusFound)
}

// Page handles GET /docs/{doc_path...}.
func (h *Handlers) Page(w http.ResponseWriter, r *http.Request) {
	docPath := NormalizePath(strings.TrimPrefix(r.URL.Path, "/"))
	page, ok := currentSite.Page(docPath)
	if !ok {
		http.NotFound(w, r)
		return
	}

	h.renderPage(w, r, page, currentSite.NavSections)
}

func (h *Handlers) renderPage(w http.ResponseWriter, r *http.Request, page Page, navSections []NavSection) {
	if err := PageView(page, navSections).Render(r.Context(), w); err != nil {
		h.logger.ErrorContext(r.Context(), "error rendering docs page", "error", err, "path", page.Path)
	}
}
