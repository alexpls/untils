package monitor

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/a-h/templ"
	"github.com/alexpls/untils/internal/models"
	"github.com/alexpls/untils/internal/pagination"
	"github.com/jackc/pgx/v5"
	"github.com/starfederation/datastar-go/datastar"
)

// ListChecks handles GET /app/checks
func (h *Handlers) ListChecks(w http.ResponseWriter, r *http.Request, user *models.User) {
	comp, err := h.renderChecksList(r, user)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "error rendering checks list", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if err := comp.Render(r.Context(), w); err != nil {
		h.logger.ErrorContext(r.Context(), "error rendering checks list", "error", err)
	}
}

// ListChecksEvents handles GET /app/checks/events (SSE)
func (h *Handlers) ListChecksEvents(w http.ResponseWriter, r *http.Request, user *models.User) {
	sse := datastar.NewSSE(w, r)
	ch := h.events.SubscribeUser(sse.Context(), user.ID)

	for {
		comp, err := h.renderChecksList(r, user)
		if err != nil {
			h.logger.ErrorContext(sse.Context(), "error rendering checks list", "error", err)
			return
		}
		if err := ssePatchElementTemplFragment(sse, comp, checksListFragment); err != nil {
			h.logger.ErrorContext(sse.Context(), "error sending checks list SSE patch", "error", err)
			return
		}

		select {
		case <-ch:
		case <-sse.Context().Done():
			return
		}
	}
}

func (h *Handlers) renderChecksList(r *http.Request, user *models.User) (templ.Component, error) {
	pag := pagination.PaginationFromRequest(r, 50)

	checks, err := h.service.queries.ListChecksWithMonitor(
		r.Context(),
		h.service.db,
		&models.ListChecksWithMonitorParams{
			UserID:    user.ID,
			PageSize:  int32(pag.PageSizeWithPeek()),
			RowOffset: int32(pag.Offset()),
		},
	)
	if err != nil {
		return nil, err
	}

	if len(checks) == pag.PageSizeWithPeek() {
		checks = checks[:pag.PageSize]
		pag.HasMore = true
	}

	return ChecksListPage(ChecksListData{
		Checks:     checks,
		Pagination: pag,
	}), nil
}

// ViewCheck handles GET /app/checks/{check_id}
func (h *Handlers) ViewCheck(w http.ResponseWriter, r *http.Request, user *models.User) {
	checkID := checkIDFromPath(r)
	if checkID == 0 {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	comp, err := h.renderCheckView(r.Context(), checkID, user.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		h.logger.ErrorContext(r.Context(), "error rendering check view", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if err := comp.Render(r.Context(), w); err != nil {
		h.logger.ErrorContext(r.Context(), "error rendering check view component", "error", err)
	}
}

// ViewCheckEvents handles GET /app/checks/{check_id}/events (SSE)
func (h *Handlers) ViewCheckEvents(w http.ResponseWriter, r *http.Request, user *models.User) {
	checkID := checkIDFromPath(r)
	if checkID == 0 {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	sse := datastar.NewSSE(w, r)
	ch := h.events.SubscribeUser(sse.Context(), user.ID)

	for {
		comp, err := h.renderCheckView(sse.Context(), checkID, user.ID)
		if err != nil {
			h.logger.ErrorContext(sse.Context(), "error rendering check view", "error", err)
			return
		}
		if err := sse.PatchElementTempl(comp); err != nil {
			h.logger.ErrorContext(sse.Context(), "error sending check view SSE patch", "error", err)
			return
		}

		select {
		case <-ch:
		case <-sse.Context().Done():
			return
		}
	}
}

func (h *Handlers) renderCheckView(ctx context.Context, checkID int64, userID int64) (templ.Component, error) {
	check, err := h.service.queries.GetCheckWithMonitor(ctx, h.service.db, checkID)
	if err != nil {
		return nil, err
	}

	if check.UserID != userID {
		return nil, pgx.ErrNoRows
	}

	conv, err := h.service.queries.GetLLMConversationBySourceID(ctx, h.service.db, &models.GetLLMConversationBySourceIDParams{
		SourceType: models.LlmConversationsSourceCheck,
		SourceID:   checkID,
	})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("getting conversation: %w", err)
	}

	messages := conv.Messages.Parse()
	toolCalls := conv.Messages.ExtractToolCalls()

	result, err := h.service.queries.GetMonitorResultByCheckID(ctx, h.service.db, checkID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("getting monitor result: %w", err)
	}

	return CheckViewPage(CheckViewData{
		Check:     check,
		Messages:  messages,
		ToolCalls: toolCalls,
		Result:    result,
	}), nil
}
