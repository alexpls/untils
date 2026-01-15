package monitor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/a-h/templ"
	"github.com/alexpls/untils/internal/db/models"
	"github.com/alexpls/untils/internal/validation"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/starfederation/datastar-go/datastar"
)

// Handlers contains the HTTP handlers for monitor routes
type Handlers struct {
	service *Service
	events  *DBEventHandler
	queries *models.Queries
	pool    *pgxpool.Pool
	logger  *slog.Logger
}

// NewHandlers creates a new Handlers instance
func NewHandlers(service *Service, events *DBEventHandler, queries *models.Queries, pool *pgxpool.Pool, logger *slog.Logger) *Handlers {
	return &Handlers{
		service: service,
		events:  events,
		queries: queries,
		pool:    pool,
		logger:  logger,
	}
}

// ListGet handles GET /app/monitors
func (h *Handlers) ListGet(w http.ResponseWriter, r *http.Request, user *models.User) {
	monitors, err := h.queries.ListMonitors(r.Context(), h.pool, user.ID)
	if err != nil {
		h.logger.Error("error listing monitors", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	MonitorsListPage(MonitorsListData{
		Monitors: monitors,
	}).Render(r.Context(), w)
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
		h.logger.Error("error viewing monitor", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	var comp templ.Component
	if mon.Status != models.MonitorStatusActive {
		comp, err = h.renderMonitorDraft(r.Context(), mon, user.ID, NewUpdateMonitorDraftParams(mon), nil)
	} else {
		comp, err = h.monitorComponent(r.Context(), mon, user.ID)
	}
	if err != nil {
		h.logger.Error("error rendering monitor", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	comp.Render(r.Context(), w)
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
		h.logger.Error("error getting monitor for events", "error", err)
		return
	}

	sse := datastar.NewSSE(w, r)

	ch := h.events.SubscribeMonitor(sse.Context(), mon.ID)

	for {
		select {
		case <-ch:
			mon, err = h.service.GetMonitor(sse.Context(), user.ID, monitorID)
			if err != nil {
				h.logger.Error("error refreshing monitor", "error", err)
				return
			}

			var comp templ.Component

			switch mon.Status {
			case models.MonitorStatusActive:
				data, err := h.monitorViewData(sse.Context(), mon, user.ID)
				if err != nil {
					h.logger.Error("error rendering monitor view", "error", err)
					return
				}

				comp = MonitorView(data)

			default:
				data, err := h.monitorDraftViewData(sse.Context(), mon, user.ID, NewUpdateMonitorDraftParams(mon), nil)
				if err != nil {
					h.logger.Error("error rendering monitor draft view", "error", err)
					return
				}

				comp = MonitorDraftView(data)
			}

			if err := sse.PatchElementTempl(comp); err != nil {
				h.logger.Error("error sending monitor view events SSE patch", "error", err)
			}

		case <-sse.Context().Done():
			return
		}
	}
}

// NewGet handles GET /app/monitors/new
func (h *Handlers) NewGet(w http.ResponseWriter, r *http.Request, user *models.User) {
	MonitorNewPage(MonitorNewData{}).Render(r.Context(), w)
}

// UpdatePost handles POST /app/monitors/{id}
func (h *Handlers) UpdatePost(w http.ResponseWriter, r *http.Request, user *models.User) {
	monitorID := monitorIDFromPath(r)
	if monitorID == 0 {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		h.logger.Error("error parsing form", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	mon, err := h.service.GetMonitor(r.Context(), user.ID, monitorID)
	if errors.Is(err, ErrMonitorNotFound) {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if err != nil {
		h.logger.Error("error updating monitor", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	monitorDraftParams := UpdateMonitorDraftParams{
		CommonParams: CommonParams{
			Subject: r.FormValue("Subject"),
		},
	}

	updatedMon, err := h.service.UpdateMonitorDraft(r.Context(), user.ID, mon.ID, monitorDraftParams)
	if err != nil {
		if validationErrs := validation.MapValidationErrors(err); validationErrs != nil {
			h.logger.Warn("failed validation when updating monitor", "validation_errors", validationErrs)
			comp, err := h.renderMonitorDraft(r.Context(), mon, user.ID, monitorDraftParams, validationErrs)
			if err != nil {
				h.logger.Error("error rendering monitor draft", "error", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
			comp.Render(r.Context(), w)
			return
		}
		h.logger.Error("error updating monitor", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	comp, err := h.renderMonitorDraft(r.Context(), updatedMon, user.ID, NewUpdateMonitorDraftParams(updatedMon), nil)
	if err != nil {
		h.logger.Error("error rendering monitor draft", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	comp.Render(r.Context(), w)
}

// ActivatePost handles POST /app/monitors/{id}/activate
func (h *Handlers) ActivatePost(w http.ResponseWriter, r *http.Request, user *models.User) {
	monitorID := monitorIDFromPath(r)
	if monitorID == 0 {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	sse := datastar.NewSSE(w, r)

	activatedMonitor, err := h.service.ActivateMonitorFromPreview(r.Context(), user.ID, monitorID)
	if err != nil {
		h.logger.Error("error activating monitor", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	sse.Redirectf("/app/monitors/%d", activatedMonitor.ID)
}

// CreatePost handles POST /app/monitors/new
func (h *Handlers) CreatePost(w http.ResponseWriter, r *http.Request, user *models.User) {
	if err := r.ParseForm(); err != nil {
		h.logger.Error("error parsing form", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	newMonitor := CreateMonitorParams{
		UserID: user.ID,
		CommonParams: CommonParams{
			Subject: r.FormValue("Subject"),
		},
	}

	createdMonitor, err := h.service.CreateMonitor(r.Context(), newMonitor)
	if err != nil {
		if validationErrs := validation.MapValidationErrors(err); validationErrs != nil {
			h.logger.Info("failed validation when creating monitor", "validation_errors", validationErrs)
			MonitorNewPage(MonitorNewData{
				ValidationErrors: validationErrs,
				Values:           newMonitor,
			}).Render(r.Context(), w)
			return
		}
		h.logger.Error("error creating monitor", "error", err)
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
		h.logger.Error("error checking monitor", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	sse := datastar.NewSSE(w, r)

	_, err = h.service.ScheduleMonitorCheck(r.Context(), mon, time.Now())
	if err != nil {
		h.logger.Error("error scheduling monitor check", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	comp, err := h.monitorComponent(r.Context(), mon, user.ID)
	if err != nil {
		h.logger.Error("error rendering monitor component", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if err := sse.PatchElementTempl(comp); err != nil {
		h.logger.Error("error patching element", "error", err)
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
		h.logger.Error("error deleting monitor", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	sse.Redirect("/app/monitors")
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
		h.logger.Error("error creating notifier", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	sse := datastar.NewSSE(w, r)

	_, err = h.service.CreateMonitorNotifier(r.Context(), mon, models.Notifier(notifierType))
	if err != nil {
		h.logger.Error("error creating notifier", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	comp, err := h.monitorNotifiersComponent(r.Context(), mon, user.ID)
	if err != nil {
		h.logger.Error("error rendering notifiers", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if err := sse.PatchElementTempl(comp); err != nil {
		h.logger.Error("error patching element", "error", err)
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
		h.logger.Error("error deleting notifier", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	sse := datastar.NewSSE(w, r)

	err = h.service.DeleteMonitorNotifier(r.Context(), mon, models.Notifier(notifierType))
	if err != nil {
		h.logger.Error("error deleting notifier", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	comp, err := h.monitorNotifiersComponent(r.Context(), mon, user.ID)
	if err != nil {
		h.logger.Error("error rendering notifiers", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if err := sse.PatchElementTempl(comp); err != nil {
		h.logger.Error("error patching element", "error", err)
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

	result, err := h.queries.GetMonitorResult(r.Context(), h.pool, &models.GetMonitorResultParams{
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
	}

	comp := monitorResultFeedback(data)
	sse.PatchElementTempl(comp, datastar.WithSelector("body"), datastar.WithModeAppend())
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

	result, err := h.queries.GetMonitorResult(r.Context(), h.pool, &models.GetMonitorResultParams{
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
			sse.PatchElementTempl(comp)

			return
		}

		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	sse := datastar.NewSSE(w, r)
	sse.Redirect(fmt.Sprintf("/app/monitors/%d", mon.ID))
}

// monitorIDFromPath extracts monitor ID from the path
func monitorIDFromPath(r *http.Request) int64 {
	monitorID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		return 0
	}
	return monitorID
}

// resultIDFromPath extracts result ID from the path
func resultIDFromPath(r *http.Request) int64 {
	resultID, err := strconv.ParseInt(r.PathValue("result_id"), 10, 64)
	if err != nil {
		return 0
	}
	return resultID
}

// Helper methods

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
		res, err := h.queries.ListMonitorResults(ctx, h.pool, mon.ID)
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

	var checkEvents []*models.MonitorCheckEvent
	if check != nil {
		var err error
		checkEvents, err = h.queries.ListMonitorCheckEvents(ctx, h.pool, check.ID)
		if err != nil {
			return MonitorDraftData{}, err
		}
	}

	notifiers, err := h.monitorNotifierViewData(ctx, mon, userID)
	if err != nil {
		return MonitorDraftData{}, err
	}

	return MonitorDraftData{
		Monitor:               mon,
		Values:                values,
		ResultPreview:         preview,
		CheckInProgress:       check,
		CheckInProgressEvents: checkEvents,
		ValidationErrors:      validationErrs,
		Notifiers:             notifiers,
	}, nil
}

func (h *Handlers) monitorViewData(ctx context.Context, mon *models.Monitor, userID int64) (MonitorViewData, error) {
	results, err := h.queries.ListMonitorResults(ctx, h.pool, mon.ID)
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

	var events []*models.MonitorCheckEvent
	if inProgressCheck != nil {
		events, err = h.queries.ListMonitorCheckEvents(ctx, h.pool, inProgressCheck.ID)
		if err != nil {
			return MonitorViewData{}, err
		}
	}

	notifiers, err := h.monitorNotifierViewData(ctx, mon, userID)
	if err != nil {
		return MonitorViewData{}, err
	}

	return MonitorViewData{
		Monitor:               mon,
		Results:               results,
		NextScheduledCheck:    nextScheduled,
		InProgressCheck:       inProgressCheck,
		InProgressCheckEvents: events,
		Notifiers:             notifiers,
	}, nil
}

func (h *Handlers) monitorNotifierViewData(ctx context.Context, mon *models.Monitor, userID int64) (notifiers []*MonitorNotifierViewData, err error) {
	notifs, err := h.service.ListMonitorNotifiers(ctx, mon)
	if err != nil {
		return notifiers, err
	}

	integrations, err := h.queries.UserIntegrations(ctx, h.pool, userID)
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
