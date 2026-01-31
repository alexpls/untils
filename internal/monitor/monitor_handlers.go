package monitor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/a-h/templ"
	"github.com/alexpls/untils/internal/models"
	"github.com/alexpls/untils/internal/pagination"
	"github.com/alexpls/untils/internal/validation"
	"github.com/jackc/pgx/v5"
	"github.com/starfederation/datastar-go/datastar"
)

// ListMonitors handles GET /app/monitors
func (h *Handlers) ListMonitors(w http.ResponseWriter, r *http.Request, user *models.User) {
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

// ListMonitorsEvents handles GET /app/monitors/events (SSE)
func (h *Handlers) ListMonitorsEvents(w http.ResponseWriter, r *http.Request, user *models.User) {
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

// ViewMonitor handles GET /app/monitors/{id}
func (h *Handlers) ViewMonitor(w http.ResponseWriter, r *http.Request, user *models.User) {
	mon := h.monitorFromPath(w, r, user)
	if mon == nil {
		return
	}

	var err error
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

// ViewMonitorEvents handles GET /app/monitors/{id}/events (SSE)
func (h *Handlers) ViewMonitorEvents(w http.ResponseWriter, r *http.Request, user *models.User) {
	mon := h.monitorFromPath(w, r, user)
	if mon == nil {
		return
	}

	sse := datastar.NewSSE(w, r)

	ch := h.events.SubscribeMonitor(sse.Context(), mon.ID)

	var err error
	for {
		mon, err = h.service.GetMonitor(sse.Context(), user.ID, mon.ID)
		if err != nil {
			h.logger.ErrorContext(sse.Context(), "error refreshing monitor", "error", err)
			return
		}

		var comp templ.Component
		if mon.Status == models.MonitorStatusActive || mon.Status == models.MonitorStatusPaused {
			comp, err = h.monitorComponent(r.Context(), mon, user.ID)
		} else {
			comp, err = h.renderMonitorDraft(r.Context(), mon, user.ID, NewUpdateMonitorDraftParams(mon), nil)
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

// ViewMonitorCheck handles GET /app/monitors/{id}/checks
func (h *Handlers) ViewMonitorCheck(w http.ResponseWriter, r *http.Request, user *models.User) {
	mon := h.monitorFromPath(w, r, user)
	if mon == nil {
		return
	}

	checks, err := h.service.queries.ListMonitorChecks(r.Context(), h.service.db, mon.ID)
	if err != nil {
		h.logger.Error("error listing monitor checks", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	comp := MonitorChecksPage(MonitorChecksViewData{
		Monitor: mon,
		Checks:  checks,
	})
	if err := comp.Render(r.Context(), w); err != nil {
		h.logger.ErrorContext(r.Context(), "error rendering monitor checks component", "error", err)
		return
	}
}

// ViewMonitorSchedule handles GET /app/monitors/{id}/schedule
func (h *Handlers) ViewMonitorSchedule(w http.ResponseWriter, r *http.Request, user *models.User) {
	mon := h.monitorFromPath(w, r, user)
	if mon == nil {
		return
	}

	comp := MonitorSchedulePage(MonitorScheduleViewData{
		Monitor: mon,
	})
	if err := comp.Render(r.Context(), w); err != nil {
		h.logger.ErrorContext(r.Context(), "error rendering monitor schedule page", "error", err)
		return
	}
}

// NewMonitor handles GET /app/monitors/new
func (h *Handlers) NewMonitor(w http.ResponseWriter, r *http.Request, user *models.User) {
	if err := MonitorNewPage(MonitorNewData{}).Render(r.Context(), w); err != nil {
		h.logger.ErrorContext(r.Context(), "error rendering monitor new page", "error", err)
	}
}

type updateCheckScheduleSignals struct {
	Schedule string `json:"schedule"`
}

// UpdateCheckSchedule handles POST /app/monitors/{id}/schedule
func (h *Handlers) UpdateCheckSchedule(w http.ResponseWriter, r *http.Request, user *models.User) {
	mon := h.monitorFromPath(w, r, user)
	if mon == nil {
		return
	}

	var params updateCheckScheduleSignals
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		h.logger.Error("error unmarshaling json", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if err := r.Body.Close(); err != nil {
		h.logger.Error("error closing response body", "error", err)
	}

	var err error
	if mon, err = h.service.UpdateMonitorSchedule(r.Context(), mon, UpdateMonitorScheduleParams{
		CheckSchedule: params.Schedule,
	}); err != nil {
		h.logger.Error("error updating monitor schedule", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	sse := datastar.NewSSE(w, r)
	comp := MonitorScheduleView(MonitorScheduleViewData{Monitor: mon})
	if err := sse.PatchElementTempl(comp, datastar.WithViewTransitions()); err != nil {
		h.logger.Error("error patching monitor schedule view", "error", err)
		return
	}
}

// UpdateMonitor handles POST /app/monitors/{id}
func (h *Handlers) UpdateMonitor(w http.ResponseWriter, r *http.Request, user *models.User) {
	mon := h.monitorFromPath(w, r, user)
	if mon == nil {
		return
	}

	var params MonitorCommonParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		h.logger.ErrorContext(r.Context(), "error decoding json", "error", err)
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

// ActivateMonitor handles POST /app/monitors/{id}/activate
func (h *Handlers) ActivateMonitor(w http.ResponseWriter, r *http.Request, user *models.User) {
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

// CreateMonitor handles POST /app/monitors/new
func (h *Handlers) CreateMonitor(w http.ResponseWriter, r *http.Request, user *models.User) {
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

// CreateMonitorCheck handles POST /app/monitors/{id}/check
func (h *Handlers) CreateMonitorCheck(w http.ResponseWriter, r *http.Request, user *models.User) {
	mon := h.monitorFromPath(w, r, user)
	if mon == nil {
		return
	}

	sse := datastar.NewSSE(w, r)

	var err error
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

// PauseMonitor handles POST /app/monitors/{id}/pause
func (h *Handlers) PauseMonitor(w http.ResponseWriter, r *http.Request, user *models.User) {
	h.setMonitorPaused(w, r, user, true)
}

// UnpauseMonitor handles POST /app/monitors/{id}/unpause
func (h *Handlers) UnpauseMonitor(w http.ResponseWriter, r *http.Request, user *models.User) {
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

// DeleteMonitor handles DELETE /app/monitors/{id}
func (h *Handlers) DeleteMonitor(w http.ResponseWriter, r *http.Request, user *models.User) {
	monitorID := monitorIDFromPath(r)
	if monitorID == 0 {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	sse := datastar.NewSSE(w, r)

	if err := h.service.DeleteMonitor(r.Context(), user.ID, monitorID); err != nil {
		h.logger.ErrorContext(r.Context(), "error deleting monitor", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if err := sse.Redirect("/app/monitors"); err != nil {
		h.logger.ErrorContext(sse.Context(), "error redirecting", "error", err)
	}
}

// UpdateMonitorNotifier handles POST /app/monitors/{id}/notifiers/{type}
func (h *Handlers) UpdateMonitorNotifier(w http.ResponseWriter, r *http.Request, user *models.User) {
	mon := h.monitorFromPath(w, r, user)
	if mon == nil {
		return
	}

	notifierType := r.PathValue("type")

	sse := datastar.NewSSE(w, r)

	if _, err := h.service.CreateMonitorNotifier(r.Context(), mon, models.Notifier(notifierType)); err != nil {
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

// DeleteMonitorNotifier handles DELETE /app/monitors/{id}/notifiers/{type}
func (h *Handlers) DeleteMonitorNotifier(w http.ResponseWriter, r *http.Request, user *models.User) {
	mon := h.monitorFromPath(w, r, user)
	if mon == nil {
		return
	}

	notifierType := r.PathValue("type")

	sse := datastar.NewSSE(w, r)

	if err := h.service.DeleteMonitorNotifier(r.Context(), mon, models.Notifier(notifierType)); err != nil {
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

// ViewResultFeedbackModal handles GET /app/monitors/{id}/results/{result_id}/feedback
func (h *Handlers) ViewResultFeedbackModal(w http.ResponseWriter, r *http.Request, user *models.User) {
	mon := h.monitorFromPath(w, r, user)
	if mon == nil {
		return
	}

	resultID := resultIDFromPath(r)

	if resultID == 0 {
		http.Error(w, "Bad request", http.StatusBadRequest)
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

// UpdateResultFeedback handles POST /app/monitors/{id}/results/{result_id}/feedback
func (h *Handlers) UpdateResultFeedback(w http.ResponseWriter, r *http.Request, user *models.User) {
	// TODO: consolidate this and ResultFeedbackGet - so much of the handler is duplicated
	mon := h.monitorFromPath(w, r, user)
	if mon == nil {
		return
	}

	resultID := resultIDFromPath(r)

	if resultID == 0 {
		http.Error(w, "Bad request", http.StatusBadRequest)
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

	if err := h.service.CreateMonitorResultFeedback(r.Context(), user.ID, result, params); err != nil {
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

func (h *Handlers) monitorFromPath(w http.ResponseWriter, r *http.Request, user *models.User) *models.Monitor {
	monitorID := monitorIDFromPath(r)
	if monitorID == 0 {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return nil
	}

	mon, err := h.service.GetMonitor(r.Context(), user.ID, monitorID)
	if err != nil {
		if errors.Is(err, ErrMonitorNotFound) {
			http.Error(w, "Not found", http.StatusNotFound)
			return nil
		}

		h.logger.ErrorContext(r.Context(), "error getting monitor", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return nil
	}

	return mon
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
	return MonitorPage(data), nil
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
