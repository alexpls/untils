package monitor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/a-h/templ"
	"github.com/alexpls/untils/internal/errortypes"
	"github.com/alexpls/untils/internal/models"
	"github.com/alexpls/untils/internal/pagination"
	"github.com/alexpls/untils/internal/validation"
	"github.com/starfederation/datastar-go/datastar"
)

const (
	monitorListPageSize     = 30
	monitorActivityPageSize = 30
	monitorChecksPageSize   = 30
)

// ListMonitors handles GET /app/monitors
func (h *Handlers) ListMonitors(w http.ResponseWriter, r *http.Request, user *models.User) {
	patcher := ConditionalPatchRenderer{
		Logger:  h.logger,
		Updater: func(ctx context.Context) (<-chan struct{}, error) { return h.events.SubscribeUser(ctx, user.ID), nil },
		Renderer: func(patch bool) (templ.Component, error) {
			pag := pagination.PaginationFromRequest(r, monitorListPageSize)

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

			monitors, pag = pagination.Peek(monitors, pag)

			data := MonitorsListData{
				Monitors:   monitors,
				Pagination: pag,
			}

			if patch {
				return MonitorsList(data), nil
			}
			return MonitorsListPage(data), nil
		},
	}
	patcher.Handle(w, r)
}

// ViewMonitor handles GET /app/monitors/{id}
func (h *Handlers) ViewMonitor(w http.ResponseWriter, r *http.Request, user *models.User) {
	monitorID := monitorIDFromPath(r)
	if monitorID == 0 {
		http.NotFound(w, r)
		return
	}

	patcher := ConditionalPatchRenderer{
		Logger: h.logger,
		Renderer: func(patch bool) (templ.Component, error) {
			freshMon, err := h.service.GetMonitor(r.Context(), user.ID, monitorID)
			if err != nil {
				return nil, err
			}

			if freshMon.Status == models.MonitorStatusActive || freshMon.Status == models.MonitorStatusPaused {
				return h.monitorComponent(r.Context(), r, freshMon, user.ID)
			}
			return h.renderMonitorDraft(r.Context(), freshMon, user.ID, NewUpdateMonitorDraftParams(freshMon), nil)
		},
		Updater: func(ctx context.Context) (<-chan struct{}, error) {
			return h.events.SubscribeMonitor(ctx, monitorID), nil
		},
	}
	patcher.Handle(w, r)
}

// ViewMonitorChecks handles GET /app/monitors/{id}/checks
func (h *Handlers) ViewMonitorChecks(w http.ResponseWriter, r *http.Request, user *models.User) {
	patcher := ConditionalPatchRenderer{
		Logger: h.logger,
		Renderer: func(patch bool) (templ.Component, error) {
			mon := h.monitorFromPath(w, r, user)
			if mon == nil {
				return nil, fmt.Errorf("monitor not found")
			}

			pag := pagination.PaginationFromRequest(r, monitorChecksPageSize)

			checks, err := h.service.queries.ListMonitorChecks(r.Context(), h.service.db, &models.ListMonitorChecksParams{
				MonitorID: mon.ID,
				PageSize:  int32(pag.PageSizeWithPeek()),
				RowOffset: int32(pag.Offset()),
			})
			if err != nil {
				return nil, err
			}

			checks, pag = pagination.Peek(checks, pag)

			data := MonitorChecksViewData{
				Monitor:    mon,
				Checks:     checks,
				Pagination: pag,
			}

			if patch {
				return MonitorChecksView(data), nil
			} else {
				return MonitorChecksPage(data), nil
			}
		},
		Updater: func(ctx context.Context) (<-chan struct{}, error) {
			monitorID := monitorIDFromPath(r)
			if monitorID == 0 {
				return nil, fmt.Errorf("monitor not found")
			}
			return h.events.SubscribeMonitor(ctx, monitorID), nil
		},
	}

	patcher.Handle(w, r)
}

// ViewMonitorNotifications handles GET /app/monitors/{id}/notifications
func (h *Handlers) ViewMonitorNotifications(w http.ResponseWriter, r *http.Request, user *models.User) {
	mon := h.monitorFromPath(w, r, user)
	if mon == nil {
		return
	}

	data, err := h.monitorNotificationsViewData(r.Context(), mon, user)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "error getting monitor notifications view data", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	comp := MonitorNotificationsPage(data)
	if err := comp.Render(r.Context(), w); err != nil {
		h.logger.ErrorContext(r.Context(), "error rendering monitor notifications page", "error", err)
		return
	}
}

// ViewMonitorSettings handles GET /app/monitors/{id}/settings
func (h *Handlers) ViewMonitorSettings(w http.ResponseWriter, r *http.Request, user *models.User) {
	mon := h.monitorFromPath(w, r, user)
	if mon == nil {
		return
	}

	data := MonitorSettingsViewData{
		Monitor: mon,
	}

	comp := MonitorSettingsPage(data)
	if err := comp.Render(r.Context(), w); err != nil {
		h.logger.ErrorContext(r.Context(), "error rendering monitor settings page", "error", err)
		return
	}
}

// NewMonitor handles GET /app/monitors/new
func (h *Handlers) NewMonitor(w http.ResponseWriter, r *http.Request, user *models.User) {
	if err := MonitorNewPage(MonitorNewData{}).Render(r.Context(), w); err != nil {
		h.logger.ErrorContext(r.Context(), "error rendering monitor new page", "error", err)
	}
}

type updateCheckFrequencySignals struct {
	Frequency int32 `json:"frequency"`
}

// UpdateMonitorCheckFrequency handles POST /app/monitors/{id}/frequency
func (h *Handlers) UpdateMonitorCheckFrequency(w http.ResponseWriter, r *http.Request, user *models.User) {
	mon := h.monitorFromPath(w, r, user)
	if mon == nil {
		return
	}

	var signals updateCheckFrequencySignals
	if err := json.NewDecoder(r.Body).Decode(&signals); err != nil {
		h.logger.Error("error unmarshaling json", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if err := r.Body.Close(); err != nil {
		h.logger.ErrorContext(r.Context(), "error closing response body", "error", err)
	}

	if _, err := h.service.UpdateMonitorCheckFrequency(r.Context(), mon, UpdateMonitorFrequencyParams{
		CheckFrequencyMinutes: signals.Frequency,
	}); err != nil {
		h.logger.ErrorContext(r.Context(), "error updating monitor frequency", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handlers) UpdateMonitorToggleAutoActivate(w http.ResponseWriter, r *http.Request, user *models.User) {
	mon := h.monitorFromPath(w, r, user)
	if mon == nil {
		return
	}

	updatedMon, err := h.service.queries.UpdateMonitorToggleAutoActivate(r.Context(), h.service.db, mon.ID)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "error updating monitor to toggle auto activate", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	sse := datastar.NewSSE(w, r)
	if err := sse.PatchElementTempl(MonitorAutoActivateForm(updatedMon)); err != nil {
		h.logger.ErrorContext(sse.Context(), "error patching monitor auto activate form", "error", err)
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

	if _, err := h.service.SetMonitorPaused(r.Context(), user, monitorID, paused); err != nil {
		h.logger.ErrorContext(r.Context(), "error setting monitor paused state", "error", err, "paused", paused)
		_ = errortypes.HandleError(err, w)
		return
	}

	if err := sseReload(sse); err != nil {
		h.logger.ErrorContext(sse.Context(), "error sending reload over sse", "error", err)
		return
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

	if _, err := h.service.CreateMonitorNotifier(r.Context(), mon, models.Notifier(notifierType)); err != nil {
		h.logger.ErrorContext(r.Context(), "error creating notifier", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	sse := datastar.NewSSE(w, r)
	h.patchMonitorNotificationsList(sse, mon, user)
}

// DeleteMonitorNotifier handles DELETE /app/monitors/{id}/notifiers/{type}
func (h *Handlers) DeleteMonitorNotifier(w http.ResponseWriter, r *http.Request, user *models.User) {
	mon := h.monitorFromPath(w, r, user)
	if mon == nil {
		return
	}

	notifierType := r.PathValue("type")

	if err := h.service.DeleteMonitorNotifier(r.Context(), mon, models.Notifier(notifierType)); err != nil {
		h.logger.ErrorContext(r.Context(), "error deleting notifier", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	sse := datastar.NewSSE(w, r)
	h.patchMonitorNotificationsList(sse, mon, user)
}

func (h *Handlers) patchMonitorNotificationsList(sse *datastar.ServerSentEventGenerator, mon *models.Monitor, user *models.User) {
	data, err := h.monitorNotificationsViewData(sse.Context(), mon, user)
	if err != nil {
		h.logger.ErrorContext(sse.Context(), "error getting monitor notifications view data", "error", err)
		return
	}

	comp := MonitorNotificationsList(data)
	if err := sse.PatchElementTempl(comp); err != nil {
		h.logger.ErrorContext(sse.Context(), "error patching element", "error", err)
		return
	}
}

// ViewResultCorrectionModal handles GET /app/monitors/{id}/results/{result_id}/correction
func (h *Handlers) ViewResultCorrectionModal(w http.ResponseWriter, r *http.Request, user *models.User) {
	mon := h.monitorFromPath(w, r, user)
	if mon == nil {
		return
	}

	result := h.monitorResultFromPath(w, r, mon)
	if result == nil {
		return
	}

	if err := h.service.AssertMonitorResultCorrectionAllowed(r.Context(), result); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	sse := datastar.NewSSE(w, r)

	data := monitorResultCorrectionViewData{
		result: result,
		formValues: CreateMonitorResultCorrectionParams{
			Correction: result.Correction.String,
		},
	}

	comp := monitorResultCorrection(data)
	if err := sse.PatchElementTempl(comp, datastar.WithSelector("body"), datastar.WithModeAppend()); err != nil {
		h.logger.ErrorContext(sse.Context(), "error patching element", "error", err)
	}
}

// UpdateResultCorrection handles POST /app/monitors/{id}/results/{result_id}/correction
func (h *Handlers) UpdateResultCorrection(w http.ResponseWriter, r *http.Request, user *models.User) {
	mon := h.monitorFromPath(w, r, user)
	if mon == nil {
		return
	}

	result := h.monitorResultFromPath(w, r, mon)
	if result == nil {
		return
	}

	var params CreateMonitorResultCorrectionParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if err := h.service.CreateMonitorResultCorrection(r.Context(), user.ID, result, params); err != nil {
		if valErrs := validation.MapValidationErrors(err); valErrs != nil {
			sse := datastar.NewSSE(w, r)

			data := monitorResultCorrectionViewData{
				result:           result,
				formValues:       params,
				validationErrors: valErrs,
			}

			comp := monitorResultCorrectionForm(data)
			if err := sse.PatchElementTempl(comp); err != nil {
				h.logger.ErrorContext(sse.Context(), "error patching element", "error", err)
			}

			return
		}

		if errors.Is(err, ErrMonitorResultCorrectionNotAllowed) {
			http.Error(w, err.Error(), http.StatusConflict)
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

// HideResult handles POST /app/monitors/{id}/results/{result_id}/hide
func (h *Handlers) HideResult(w http.ResponseWriter, r *http.Request, user *models.User) {
	mon := h.monitorFromPath(w, r, user)
	if mon == nil {
		return
	}

	result := h.monitorResultFromPath(w, r, mon)
	if result == nil {
		return
	}

	if err := h.service.HideMonitorResult(r.Context(), user.ID, result); err != nil {
		if errors.Is(err, ErrMonitorResultHideNotAllowed) {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}

		_ = errortypes.HandleError(err, w)
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
		http.NotFound(w, r)
		return nil
	}

	mon, err := h.service.GetMonitor(r.Context(), user.ID, monitorID)
	if err != nil {
		_ = errortypes.HandleError(err, w)
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

func (h *Handlers) monitorComponent(ctx context.Context, r *http.Request, mon *models.Monitor, userID int64) (templ.Component, error) {
	data, err := h.monitorViewData(ctx, r, mon, userID)
	if err != nil {
		return nil, err
	}
	return MonitorPage(data), nil
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
		// Ignore not-found cases - events may not exist yet.
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

func (h *Handlers) monitorViewData(ctx context.Context, r *http.Request, mon *models.Monitor, userID int64) (MonitorViewData, error) {
	pag := pagination.PaginationFromRequest(r, monitorActivityPageSize)

	results, err := h.service.queries.ListMonitorResultsWithLatestCheck(ctx, h.service.db, &models.ListMonitorResultsWithLatestCheckParams{
		MonitorID: mon.ID,
		PageSize:  int32(pag.PageSizeWithPeek()),
		RowOffset: int32(pag.Offset()),
	})
	if err != nil {
		return MonitorViewData{}, err
	}
	results, pag = pagination.Peek(results, pag)

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
		// Ignore not-found cases - events may not exist yet.
	}

	notifiers, err := h.monitorNotifierViewData(ctx, mon, userID)
	if err != nil {
		return MonitorViewData{}, err
	}

	return MonitorViewData{
		Monitor:                       mon,
		Results:                       results,
		Pagination:                    pag,
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

func (h *Handlers) monitorNotificationsViewData(ctx context.Context, mon *models.Monitor, user *models.User) (MonitorNotificationsViewData, error) {
	notifiers, err := h.monitorNotifierViewData(ctx, mon, user.ID)
	if err != nil {
		return MonitorNotificationsViewData{}, err
	}

	return MonitorNotificationsViewData{
		Monitor:   mon,
		Notifiers: notifiers,
	}, nil
}
