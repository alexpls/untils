package monitor

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/a-h/templ"
	"github.com/alexpls/untils/internal/models"
	"github.com/alexpls/untils/internal/pagination"
	"github.com/alexpls/untils/internal/validation"
	"github.com/jackc/pgx/v5"
	"github.com/starfederation/datastar-go/datastar"
)

// Handlers contains the HTTP handlers for monitor routes
type Handlers struct {
	service *Service
	events  *DBEventHandler
	logger  *slog.Logger
}

// NewHandlers creates a new Handlers instance
func NewHandlers(service *Service, events *DBEventHandler, logger *slog.Logger) *Handlers {
	return &Handlers{
		service: service,
		events:  events,
		logger:  logger,
	}
}

// ListGet handles GET /app/monitors
func (h *Handlers) ListGet(w http.ResponseWriter, r *http.Request, user *models.User) {
	comp, err := h.renderMonitorList(r, user)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "error rendering monitor list", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if err := comp.Render(r.Context(), w); err != nil {
		h.logger.ErrorContext(r.Context(), "error rendering monitors list", "error", err)
	}
}

// ListEventsGet handles GET /app/monitors/events (SSE)
func (h *Handlers) ListEventsGet(w http.ResponseWriter, r *http.Request, user *models.User) {
	sse := datastar.NewSSE(w, r)
	ch := h.events.SubscribeUser(sse.Context(), user.ID)

	for {
		comp, err := h.renderMonitorList(r, user)
		if err != nil {
			h.logger.ErrorContext(sse.Context(), "error rendering monitor list", "error", err)
			return
		}
		if err := ssePatchElementTemplFragment(sse, comp, monitorListFragment); err != nil {
			h.logger.ErrorContext(sse.Context(), "error sending monitors list SSE patch", "error", err)
			return
		}

		select {
		case <-ch:
		case <-sse.Context().Done():
			return
		}
	}
}

func (h *Handlers) renderMonitorList(r *http.Request, user *models.User) (templ.Component, error) {
	pag := pagination.PaginationFromRequest(r, 30)

	monitors, err := h.service.queries.ListMonitorsWithResults(
		r.Context(),
		h.service.db,
		&models.ListMonitorsWithResultsParams{
			UserID:    user.ID,
			PageSize:  int32(pag.PageSizeWithPeek()),
			RowOffset: int32(pag.Offset()),
		},
	)
	if err != nil {
		return nil, err
	}

	// TODO: move this peeking logic to something more generic in pagination package
	if len(monitors) == pag.PageSizeWithPeek() {
		monitors = monitors[:pag.PageSize]
		pag.HasMore = true
	}

	return MonitorsListPage(MonitorsListData{
		Monitors:   monitors,
		Pagination: pag,
	}), nil
}

// ChecksListGet handles GET /app/checks
func (h *Handlers) ChecksListGet(w http.ResponseWriter, r *http.Request, user *models.User) {
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

// ChecksListEventsGet handles GET /app/checks/events (SSE)
func (h *Handlers) ChecksListEventsGet(w http.ResponseWriter, r *http.Request, user *models.User) {
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

// CheckViewGet handles GET /app/checks/{check_id}
func (h *Handlers) CheckViewGet(w http.ResponseWriter, r *http.Request, user *models.User) {
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

// CheckViewEventsGet handles GET /app/checks/{check_id}/events (SSE)
func (h *Handlers) CheckViewEventsGet(w http.ResponseWriter, r *http.Request, user *models.User) {
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
	if err == nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("getting monitor result: %w", err)
	}

	return CheckViewPage(CheckViewData{
		Check:     check,
		Messages:  messages,
		ToolCalls: toolCalls,
		Result:    result,
	}), nil
}

// ViewGet handles GET /app/monitors/{id}
func (h *Handlers) ViewGet(w http.ResponseWriter, r *http.Request, user *models.User) {
	monitorID := monitorIDFromPath(r)
	if monitorID == 0 {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	mon, err := h.service.GetMonitor(r.Context(), user.ID, monitorID)
	if errors.Is(err, ErrMonitorNotFound) {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if err != nil {
		h.logger.ErrorContext(r.Context(), "error viewing monitor", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	var comp templ.Component
	if mon.Status == models.MonitorStatusActive || mon.Status == models.MonitorStatusPaused {
		comp, err = h.monitorComponent(r.Context(), mon, user.ID)
	} else {
		comp, err = h.renderMonitorDraft(r.Context(), mon, user.ID, NewUpdateMonitorDraftParams(mon), nil)
	}
	if err != nil {
		h.logger.ErrorContext(r.Context(), "error rendering monitor", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if err := comp.Render(r.Context(), w); err != nil {
		h.logger.ErrorContext(r.Context(), "error rendering monitor component", "error", err)
	}
}

// ViewEventsGet handles GET /app/monitors/{id}/events (SSE)
func (h *Handlers) ViewEventsGet(w http.ResponseWriter, r *http.Request, user *models.User) {
	monitorID := monitorIDFromPath(r)
	if monitorID == 0 {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	mon, err := h.service.GetMonitor(r.Context(), user.ID, monitorID)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "error getting monitor for events", "error", err)
		return
	}

	sse := datastar.NewSSE(w, r)

	ch := h.events.SubscribeMonitor(sse.Context(), mon.ID)

	for {
		mon, err = h.service.GetMonitor(sse.Context(), user.ID, monitorID)
		if err != nil {
			h.logger.ErrorContext(sse.Context(), "error refreshing monitor", "error", err)
			return
		}

		var comp templ.Component

		switch mon.Status {
		case models.MonitorStatusActive, models.MonitorStatusPaused:
			data, err := h.monitorViewData(sse.Context(), mon, user.ID)
			if err != nil {
				h.logger.ErrorContext(sse.Context(), "error rendering monitor view", "error", err)
				return
			}

			comp = MonitorView(data)

		default:
			data, err := h.monitorDraftViewData(sse.Context(), mon, user.ID, NewUpdateMonitorDraftParams(mon), nil)
			if err != nil {
				h.logger.ErrorContext(sse.Context(), "error rendering monitor draft view", "error", err)
				return
			}

			comp = MonitorDraftView(data)
		}

		if err := sse.PatchElementTempl(comp); err != nil {
			h.logger.ErrorContext(sse.Context(), "error sending monitor view events SSE patch", "error", err)
		}

		select {
		case <-ch:
		case <-sse.Context().Done():
			return
		}
	}
}

// NewGet handles GET /app/monitors/new
func (h *Handlers) NewGet(w http.ResponseWriter, r *http.Request, user *models.User) {
	if err := MonitorNewPage(MonitorNewData{}).Render(r.Context(), w); err != nil {
		h.logger.ErrorContext(r.Context(), "error rendering monitor new page", "error", err)
	}
}

// UpdatePost handles POST /app/monitors/{id}
func (h *Handlers) UpdatePost(w http.ResponseWriter, r *http.Request, user *models.User) {
	monitorID := monitorIDFromPath(r)
	if monitorID == 0 {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	var params MonitorCommonParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		h.logger.ErrorContext(r.Context(), "error decoding json", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	mon, err := h.service.GetMonitor(r.Context(), user.ID, monitorID)
	if errors.Is(err, ErrMonitorNotFound) {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if err != nil {
		h.logger.ErrorContext(r.Context(), "error updating monitor", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	sse := datastar.NewSSE(w, r)

	monitorDraftParams := UpdateMonitorDraftParams{
		MonitorCommonParams: params,
	}
	viewData, err := h.monitorDraftViewData(sse.Context(), mon, user.ID, monitorDraftParams, nil)
	if err != nil {
		h.logger.ErrorContext(sse.Context(), "error making monitor draft view data", "error", err)
		return
	}

	updatedMon, err := h.service.UpdateMonitorDraft(r.Context(), user.ID, mon.ID, monitorDraftParams)
	if err != nil {
		if validationErrs := validation.MapValidationErrors(err); validationErrs != nil {
			h.logger.WarnContext(r.Context(), "failed validation when updating monitor", "validation_errors", validationErrs)

			viewData.ValidationErrors = validationErrs

			if err := sse.PatchElementTempl(MonitorDraftView(viewData)); err != nil {
				h.logger.ErrorContext(sse.Context(), "error sending monitor draft view events SSE patch", "error", err)
			}

			return
		}

		h.logger.ErrorContext(r.Context(), "error updating monitor", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	viewData.Monitor = updatedMon

	if err := sse.PatchElementTempl(MonitorDraftView(viewData)); err != nil {
		h.logger.ErrorContext(sse.Context(), "error sending monitor draft view events SSE patch", "error", err)
	}
}

// ActivatePost handles POST /app/monitors/{id}/activate
func (h *Handlers) ActivatePost(w http.ResponseWriter, r *http.Request, user *models.User) {
	monitorID := monitorIDFromPath(r)
	if monitorID == 0 {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	sse := datastar.NewSSE(w, r)

	activatedMonitor, err := h.service.ActivateMonitorFromPreview(r.Context(), user, monitorID)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "error activating monitor", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if err := sse.Redirectf("/app/monitors/%d", activatedMonitor.ID); err != nil {
		h.logger.ErrorContext(sse.Context(), "error redirecting after activating monitor", "error", err)
	}
}

// CreatePost handles POST /app/monitors/new
func (h *Handlers) CreatePost(w http.ResponseWriter, r *http.Request, user *models.User) {
	if err := r.ParseForm(); err != nil {
		h.logger.ErrorContext(r.Context(), "error parsing form", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	newMonitor := CreateMonitorParams{
		UserID: user.ID,
		MonitorCommonParams: MonitorCommonParams{
			Subject: r.FormValue("Subject"),
		},
	}

	createdMonitor, err := h.service.CreateMonitor(r.Context(), newMonitor)
	if err != nil {
		if validationErrs := validation.MapValidationErrors(err); validationErrs != nil {
			h.logger.InfoContext(r.Context(), "failed validation when creating monitor", "validation_errors", validationErrs)
			if err := MonitorNewPage(MonitorNewData{
				ValidationErrors: validationErrs,
				Values:           newMonitor,
			}).Render(r.Context(), w); err != nil {
				h.logger.ErrorContext(r.Context(), "error rendering monitor new page", "error", err)
			}
			return
		}
		h.logger.ErrorContext(r.Context(), "error creating monitor", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/app/monitors/%d", createdMonitor.ID), http.StatusSeeOther)
}

// CheckPost handles POST /app/monitors/{id}/check
func (h *Handlers) CheckPost(w http.ResponseWriter, r *http.Request, user *models.User) {
	monitorID := monitorIDFromPath(r)
	if monitorID == 0 {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	mon, err := h.service.GetMonitor(r.Context(), user.ID, monitorID)
	if errors.Is(err, ErrMonitorNotFound) {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if err != nil {
		h.logger.ErrorContext(r.Context(), "error checking monitor", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	sse := datastar.NewSSE(w, r)

	_, err = h.service.ScheduleMonitorCheck(r.Context(), mon, time.Now())
	if err != nil {
		h.logger.ErrorContext(r.Context(), "error scheduling monitor check", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	comp, err := h.monitorComponent(r.Context(), mon, user.ID)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "error rendering monitor component", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if err := sse.PatchElementTempl(comp); err != nil {
		h.logger.ErrorContext(sse.Context(), "error patching element", "error", err)
	}
}

// PausePost handles POST /app/monitors/{id}/pause
func (h *Handlers) PausePost(w http.ResponseWriter, r *http.Request, user *models.User) {
	h.setMonitorPaused(w, r, user, true)
}

// UnpausePost handles POST /app/monitors/{id}/unpause
func (h *Handlers) UnpausePost(w http.ResponseWriter, r *http.Request, user *models.User) {
	h.setMonitorPaused(w, r, user, false)
}

func (h *Handlers) setMonitorPaused(w http.ResponseWriter, r *http.Request, user *models.User, paused bool) {
	monitorID := monitorIDFromPath(r)
	if monitorID == 0 {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	sse := datastar.NewSSE(w, r)

	mon, err := h.service.SetMonitorPaused(r.Context(), user, monitorID, paused)
	if err != nil {
		if errors.Is(err, ErrMonitorNotFound) {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		var transitionErr *ErrInvalidStatusTransition
		if errors.As(err, &transitionErr) {
			http.Error(w, "Invalid monitor state transition", http.StatusBadRequest)
			return
		}
		h.logger.ErrorContext(r.Context(), "error setting monitor paused state", "error", err, "paused", paused)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	comp, err := h.monitorComponent(r.Context(), mon, user.ID)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "error rendering monitor component", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if err := sse.PatchElementTempl(comp); err != nil {
		h.logger.ErrorContext(sse.Context(), "error patching element", "error", err)
	}
}

// Delete handles DELETE /app/monitors/{id}
func (h *Handlers) Delete(w http.ResponseWriter, r *http.Request, user *models.User) {
	monitorID := monitorIDFromPath(r)
	if monitorID == 0 {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	sse := datastar.NewSSE(w, r)

	err := h.service.DeleteMonitor(r.Context(), user.ID, monitorID)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "error deleting monitor", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if err := sse.Redirect("/app/monitors"); err != nil {
		h.logger.ErrorContext(sse.Context(), "error redirecting", "error", err)
	}
}

// NotifierPost handles POST /app/monitors/{id}/notifiers/{type}
func (h *Handlers) NotifierPost(w http.ResponseWriter, r *http.Request, user *models.User) {
	monitorID := monitorIDFromPath(r)
	if monitorID == 0 {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	notifierType := r.PathValue("type")

	mon, err := h.service.GetMonitor(r.Context(), user.ID, monitorID)
	if errors.Is(err, ErrMonitorNotFound) {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if err != nil {
		h.logger.ErrorContext(r.Context(), "error creating notifier", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	sse := datastar.NewSSE(w, r)

	_, err = h.service.CreateMonitorNotifier(r.Context(), mon, models.Notifier(notifierType))
	if err != nil {
		h.logger.ErrorContext(r.Context(), "error creating notifier", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	comp, err := h.monitorNotifiersComponent(r.Context(), mon, user.ID)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "error rendering notifiers", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if err := sse.PatchElementTempl(comp); err != nil {
		h.logger.ErrorContext(sse.Context(), "error patching element", "error", err)
	}
}

// NotifierDelete handles DELETE /app/monitors/{id}/notifiers/{type}
func (h *Handlers) NotifierDelete(w http.ResponseWriter, r *http.Request, user *models.User) {
	monitorID := monitorIDFromPath(r)
	if monitorID == 0 {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	notifierType := r.PathValue("type")

	mon, err := h.service.GetMonitor(r.Context(), user.ID, monitorID)
	if errors.Is(err, ErrMonitorNotFound) {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if err != nil {
		h.logger.ErrorContext(r.Context(), "error deleting notifier", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	sse := datastar.NewSSE(w, r)

	err = h.service.DeleteMonitorNotifier(r.Context(), mon, models.Notifier(notifierType))
	if err != nil {
		h.logger.ErrorContext(r.Context(), "error deleting notifier", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	comp, err := h.monitorNotifiersComponent(r.Context(), mon, user.ID)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "error rendering notifiers", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if err := sse.PatchElementTempl(comp); err != nil {
		h.logger.ErrorContext(sse.Context(), "error patching element", "error", err)
	}
}

// ResultFeedbackGet handles GET /app/monitors/{id}/results/{result_id}/feedback
func (h *Handlers) ResultFeedbackGet(w http.ResponseWriter, r *http.Request, user *models.User) {
	monitorID := monitorIDFromPath(r)
	resultID := resultIDFromPath(r)

	if monitorID == 0 || resultID == 0 {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	mon, err := h.service.GetMonitor(r.Context(), user.ID, monitorID)
	if err != nil {
		if errors.Is(err, ErrMonitorNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	result, err := h.service.queries.GetMonitorResult(r.Context(), h.service.db, &models.GetMonitorResultParams{
		MonitorID: mon.ID,
		ResultID:  resultID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.NotFound(w, r)
			return
		}
	}

	sse := datastar.NewSSE(w, r)

	data := monitorResultFeedbackViewData{
		result: result,
		formValues: CreateMonitorResultFeedbackParams{
			Feedback: result.Feedback.String,
		},
	}

	comp := monitorResultFeedback(data)
	if err := sse.PatchElementTempl(comp, datastar.WithSelector("body"), datastar.WithModeAppend()); err != nil {
		h.logger.ErrorContext(sse.Context(), "error patching element", "error", err)
	}
}

// ResultFeedbackPost handles POST /app/monitors/{id}/results/{result_id}/feedback
func (h *Handlers) ResultFeedbackPost(w http.ResponseWriter, r *http.Request, user *models.User) {
	// TODO: consolidate this and ResultFeedbackGet - so much of the handler is duplicated

	monitorID := monitorIDFromPath(r)
	resultID := resultIDFromPath(r)

	if monitorID == 0 || resultID == 0 {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	mon, err := h.service.GetMonitor(r.Context(), user.ID, monitorID)
	if err != nil {
		if errors.Is(err, ErrMonitorNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	result, err := h.service.queries.GetMonitorResult(r.Context(), h.service.db, &models.GetMonitorResultParams{
		MonitorID: mon.ID,
		ResultID:  resultID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.NotFound(w, r)
			return
		}
	}

	var params CreateMonitorResultFeedbackParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	err = h.service.CreateMonitorResultFeedback(r.Context(), user.ID, result, params)
	if err != nil {
		if valErrs := validation.MapValidationErrors(err); valErrs != nil {
			sse := datastar.NewSSE(w, r)
			data := monitorResultFeedbackViewData{
				result:           result,
				formValues:       params,
				validationErrors: valErrs,
			}

			comp := monitorResultFeedbackForm(data)
			if err := sse.PatchElementTempl(comp); err != nil {
				h.logger.ErrorContext(sse.Context(), "error patching element", "error", err)
			}

			return
		}

		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	sse := datastar.NewSSE(w, r)
	if err := sseReload(sse); err != nil {
		h.logger.ErrorContext(sse.Context(), "error reloading", "error", err)
	}
}

// monitorIDFromPath extracts monitor ID from the path
func monitorIDFromPath(r *http.Request) int64 {
	return idFromPath(r, "monitor_id")
}

// resultIDFromPath extracts result ID from the path
func resultIDFromPath(r *http.Request) int64 {
	return idFromPath(r, "result_id")
}

// checkIDFromPath extracts check ID from the path
func checkIDFromPath(r *http.Request) int64 {
	return idFromPath(r, "check_id")
}

// idFromPath extracts an int64 ID from the path of the request
func idFromPath(r *http.Request, name string) int64 {
	id, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	if err != nil {
		return 0
	}
	return id
}

func (h *Handlers) renderMonitorDraft(
	ctx context.Context,
	mon *models.Monitor,
	userID int64,
	values UpdateMonitorDraftParams,
	validationErrs validation.ValidationErrors,
) (templ.Component, error) {
	data, err := h.monitorDraftViewData(ctx, mon, userID, values, validationErrs)
	if err != nil {
		return nil, err
	}
	return MonitorDraftPage(data), nil
}

func (h *Handlers) monitorComponent(ctx context.Context, mon *models.Monitor, userID int64) (templ.Component, error) {
	data, err := h.monitorViewData(ctx, mon, userID)
	if err != nil {
		return nil, err
	}
	return MonitorViewPage(data), nil
}

func (h *Handlers) monitorNotifiersComponent(ctx context.Context, mon *models.Monitor, userID int64) (templ.Component, error) {
	data, err := h.monitorNotifierViewData(ctx, mon, userID)
	if err != nil {
		return nil, err
	}
	return MonitorNotifiers(mon, data), nil
}

func (h *Handlers) monitorDraftViewData(
	ctx context.Context,
	mon *models.Monitor,
	userID int64,
	values UpdateMonitorDraftParams,
	validationErrs validation.ValidationErrors,
) (MonitorDraftData, error) {
	var preview *models.MonitorResult
	if mon.Status == models.MonitorStatusReady {
		res, err := h.service.queries.ListMonitorResults(ctx, h.service.db, mon.ID)
		if err != nil {
			return MonitorDraftData{}, err
		}
		if len(res) != 1 {
			return MonitorDraftData{}, fmt.Errorf("expected exactly one monitor result for preview, got %d", len(res))
		}
		preview = res[0]
	}

	check, err := h.service.GetInProgressMonitorCheck(ctx, mon)
	if err != nil {
		return MonitorDraftData{}, err
	}

	var timelineEvents []*models.GetTimelineEventsBySourceIDRow
	if check != nil {
		events, err := h.service.queries.GetTimelineEventsBySourceID(ctx, h.service.db, &models.GetTimelineEventsBySourceIDParams{
			SourceType: models.LlmConversationsSourceCheck,
			SourceID:   check.ID,
		})
		if err == nil {
			timelineEvents = events
		}
		// Ignore pgx.ErrNoRows - events may not exist yet
	}

	notifiers, err := h.monitorNotifierViewData(ctx, mon, userID)
	if err != nil {
		return MonitorDraftData{}, err
	}

	return MonitorDraftData{
		Monitor:                       mon,
		Values:                        values,
		ResultPreview:                 preview,
		InProgressCheck:               check,
		InProgressCheckTimelineEvents: timelineEvents,
		ValidationErrors:              validationErrs,
		Notifiers:                     notifiers,
	}, nil
}

func (h *Handlers) monitorViewData(ctx context.Context, mon *models.Monitor, userID int64) (MonitorViewData, error) {
	results, err := h.service.queries.ListMonitorResults(ctx, h.service.db, mon.ID)
	if err != nil {
		return MonitorViewData{}, err
	}

	nextScheduled, err := h.service.GetNextMonitorCheck(ctx, mon)
	if err != nil {
		return MonitorViewData{}, err
	}

	inProgressCheck, err := h.service.GetInProgressMonitorCheck(ctx, mon)
	if err != nil {
		return MonitorViewData{}, err
	}

	var timelineEvents []*models.GetTimelineEventsBySourceIDRow
	if inProgressCheck != nil {
		events, err := h.service.queries.GetTimelineEventsBySourceID(ctx, h.service.db, &models.GetTimelineEventsBySourceIDParams{
			SourceType: models.LlmConversationsSourceCheck,
			SourceID:   inProgressCheck.ID,
		})
		if err == nil {
			timelineEvents = events
		}
		// Ignore pgx.ErrNoRows - events may not exist yet
	}

	notifiers, err := h.monitorNotifierViewData(ctx, mon, userID)
	if err != nil {
		return MonitorViewData{}, err
	}

	return MonitorViewData{
		Monitor:                       mon,
		Results:                       results,
		NextScheduledCheck:            nextScheduled,
		InProgressCheck:               inProgressCheck,
		InProgressCheckTimelineEvents: timelineEvents,
		Notifiers:                     notifiers,
	}, nil
}

func (h *Handlers) monitorNotifierViewData(ctx context.Context, mon *models.Monitor, userID int64) (notifiers []*MonitorNotifierViewData, err error) {
	notifs, err := h.service.ListMonitorNotifiers(ctx, mon)
	if err != nil {
		return notifiers, err
	}

	integrations, err := h.service.queries.UserIntegrations(ctx, h.service.db, userID)
	if err != nil {
		return notifiers, err
	}

	for _, integration := range integrations {
		if !integration.Configured {
			continue
		}

		var active bool

		for _, notif := range notifs {
			if notif.Type == integration.Name {
				active = true
				break
			}
		}

		notifiers = append(notifiers, &MonitorNotifierViewData{
			Integration: integration,
			Active:      active,
		})
	}

	return notifiers, nil
}

// ssePatchElementTemplFragment sends HTML to the sse stream for the given templ component and fragment
func ssePatchElementTemplFragment(sse *datastar.ServerSentEventGenerator, c templ.Component, fragmentIDs ...any) error {
	var buf bytes.Buffer
	if err := templ.RenderFragments(sse.Context(), &buf, c, fragmentIDs...); err != nil {
		return fmt.Errorf("failed to patch element: %w", err)
	}
	if err := sse.PatchElements(buf.String()); err != nil {
		return fmt.Errorf("failed to patch element: %w", err)
	}
	return nil
}

// sseReload sends javascript to the sse stream to reload the page
func sseReload(sse *datastar.ServerSentEventGenerator) error {
	js := "setTimeout(() => window.location.reload())"
	return sse.ExecuteScript(js)
}
