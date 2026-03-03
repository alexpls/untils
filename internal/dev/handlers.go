package dev

import (
	"log/slog"
	"net/http"

	"github.com/alexpls/untils/internal/models"
	"github.com/alexpls/untils/internal/reqcontext"
)

type Handlers struct {
	logger *slog.Logger
}

func NewHandlers(logger *slog.Logger) *Handlers {
	return &Handlers{
		logger: logger,
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
