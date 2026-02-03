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

			result, err := h.service.queries.GetMonitorResultByCheckID(r.Context(), h.service.db, checkID)
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return nil, fmt.Errorf("getting monitor result: %w", err)
			}

			data := CheckViewData{
				Check:     check,
				Messages:  messages,
				ToolCalls: toolCalls,
				Result:    result,
			}

			if patch {
				return CheckView(data), nil
			}
			return CheckViewPage(data), nil
		},
	}
	patcher.Handle(w, r)
}
