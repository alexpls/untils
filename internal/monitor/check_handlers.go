package monitor

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/a-h/templ"
	"github.com/alexpls/untils/internal/errortypes"
	"github.com/alexpls/untils/internal/models"
	"github.com/alexpls/untils/internal/pagination"
	"github.com/jackc/pgx/v5"
)

// ListChecks handles GET /app/checks
func (h *Handlers) ListChecks(w http.ResponseWriter, r *http.Request, user *models.User) {
	patcher := ConditionalPatchRenderer{
		Logger:  h.logger,
		Updater: func(ctx context.Context) (<-chan struct{}, error) { return h.events.SubscribeUser(ctx, user.ID), nil },
		Renderer: func(patch bool) (templ.Component, error) {
			pag := pagination.PaginationFromRequest(r, 50)

			checks, err := h.service.queries.ListChecksWithMonitor(
				r.Context(),
				h.service.db,
				&models.ListChecksWithMonitorParams{
					UserID:    user.ID,
					PageSize:  int64(pag.PageSizeWithPeek()),
					RowOffset: pag.Offset64(),
				},
			)
			if err != nil {
				return nil, err
			}

			checks, pag = pagination.Peek(checks, pag)

			data := ChecksListData{
				Checks:     checks,
				Pagination: pag,
			}

			if patch {
				return ChecksList(data), nil
			}
			return ChecksListPage(data), nil
		},
	}
	patcher.Handle(w, r)
}

// ViewCheck handles GET /app/checks/{check_id}
func (h *Handlers) ViewCheck(w http.ResponseWriter, r *http.Request, user *models.User) {
	checkID := checkIDFromPath(r)
	if checkID == 0 {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	patcher := ConditionalPatchRenderer{
		Logger:  h.logger,
		Updater: func(ctx context.Context) (<-chan struct{}, error) { return h.events.SubscribeUser(ctx, user.ID), nil },
		Renderer: func(patch bool) (templ.Component, error) {
			check, err := h.service.queries.GetCheckWithMonitor(r.Context(), h.service.db, checkID)
			if err != nil {
				return nil, err
			}

			if check.UserID != user.ID {
				return nil, pgx.ErrNoRows
			}

			conv, err := h.service.queries.GetLLMConversationBySourceID(r.Context(), h.service.db, &models.GetLLMConversationBySourceIDParams{
				SourceType: models.LlmConversationsSourceCheck,
				SourceID:   checkID,
			})
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return nil, fmt.Errorf("getting conversation: %w", err)
			}

			messages := conv.Messages.Parse()
			toolCalls := conv.Messages.ExtractToolCalls()

			results, err := h.service.queries.ListMonitorResultsByCheckID(r.Context(), h.service.db, checkID)
			if err != nil {
				return nil, fmt.Errorf("getting monitor results: %w", err)
			}

			data := CheckViewData{
				Check:     check,
				Messages:  messages,
				ToolCalls: toolCalls,
				Results:   results,
			}

			if patch {
				return CheckView(data), nil
			}
			return CheckViewPage(data), nil
		},
	}
	patcher.Handle(w, r)
}

// RunCheckNow handles POST /app/checks/{check_id}/run
func (h *Handlers) RunCheckNow(w http.ResponseWriter, r *http.Request, user *models.User) {
	checkID := checkIDFromPath(r)
	if checkID == 0 {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// Verify the check belongs to this user
	check, err := h.service.queries.GetCheckWithMonitor(r.Context(), h.service.db, checkID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		h.logger.ErrorContext(r.Context(), "error getting check", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if check.UserID != user.ID {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	if err := h.service.RunCheckNow(r.Context(), checkID); err != nil {
		if errors.Is(err, &errortypes.ErrCheckNotScheduled{}) {
			http.Error(w, "Check is not scheduled", http.StatusBadRequest)
			return
		}
		h.logger.ErrorContext(r.Context(), "error running check now", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
