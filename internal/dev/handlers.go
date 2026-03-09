package dev

import (
	"log/slog"
	"net/http"

	"github.com/alexpls/untils/internal/models"
	"github.com/alexpls/untils/internal/notifications"
	"github.com/alexpls/untils/internal/reqcontext"
)

type Handlers struct {
	logger        *slog.Logger
	emailPreviews *notifications.EmailTemplateStore
}

func NewHandlers(logger *slog.Logger, emailPreviews *notifications.EmailTemplateStore) *Handlers {
	return &Handlers{
		logger:        logger,
		emailPreviews: emailPreviews,
	}
}

func (h *Handlers) ViewPalette(w http.ResponseWriter, r *http.Request, _ *models.User) {
	component := PalettePage()
	if err := component.Render(r.Context(), w); err != nil {
		h.logger.Error("error rendering palette", "error", err)
	}
}

func (h *Handlers) ViewMonitorDraftPalette(w http.ResponseWriter, r *http.Request, _ *models.User) {
	component := MonitorDraftPalettePage()
	if err := component.Render(r.Context(), w); err != nil {
		h.logger.Error("error rendering monitor draft palette", "error", err)
	}
}

func (h *Handlers) ViewFlashPalette(w http.ResponseWriter, r *http.Request, _ *models.User) {
	ctx := reqcontext.ContextWithFlashAlert(r.Context(), "Password changed")
	component := FlashPalettePage()
	if err := component.Render(ctx, w); err != nil {
		h.logger.Error("error rendering flash palette", "error", err)
	}
}

func (h *Handlers) ListEmailPreviews(w http.ResponseWriter, r *http.Request, _ *models.User) {
	component := EmailPreviewsIndexPage(h.emailPreviews.Templates())
	if err := component.Render(r.Context(), w); err != nil {
		h.logger.ErrorContext(r.Context(), "error rendering email previews index", "error", err)
	}
}

func (h *Handlers) ViewEmailPreview(w http.ResponseWriter, r *http.Request, _ *models.User) {
	templateKey := r.PathValue("template_key")
	tmpl, ok := h.emailPreviews.Template(templateKey)
	if !ok {
		http.NotFound(w, r)
		return
	}

	rendered, err := tmpl.Render(r.Context(), tmpl.DummyData)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "error rendering email preview", "template_key", templateKey, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	component := EmailPreviewPage(EmailPreviewPageData{
		Template: tmpl,
		Rendered: rendered,
		HTMLURL:  "/app/dev/emails/" + templateKey + "/html",
	})
	if err := component.Render(r.Context(), w); err != nil {
		h.logger.ErrorContext(r.Context(), "error rendering email preview page", "template_key", templateKey, "error", err)
	}
}

func (h *Handlers) ViewEmailPreviewHTML(w http.ResponseWriter, r *http.Request, _ *models.User) {
	templateKey := r.PathValue("template_key")
	tmpl, ok := h.emailPreviews.Template(templateKey)
	if !ok {
		http.NotFound(w, r)
		return
	}

	rendered, err := tmpl.Render(r.Context(), tmpl.DummyData)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "error rendering email preview html", "template_key", templateKey, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(rendered.HTMLBody))
}
