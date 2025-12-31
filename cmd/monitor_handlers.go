package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/a-h/templ"
	appcomponents "github.com/alexpls/untils_go/internal/components/app"
	"github.com/alexpls/untils_go/internal/db/sqlc"
	"github.com/alexpls/untils_go/internal/monitor"
	"github.com/alexpls/untils_go/internal/validation"
	"github.com/starfederation/datastar-go/datastar"
)

func (a *app) monitorListGet(w http.ResponseWriter, r *http.Request, u *sqlc.User) {
	monitors, err := a.monitor.ListMonitors(r.Context(), u.ID)
	if a.internalServerError(err, w) {
		a.logger.Error("error listing monitors", "error", err)
		return
	}

	appcomponents.MonitorsListPage(appcomponents.MonitorsListData{
		Monitors: monitors,
	}).Render(r.Context(), w)
}

func (a *app) renderMonitorDraft(ctx context.Context, mon *sqlc.Monitor, values monitor.UpdateMonitorDraftParams, validationErrs validation.ValidationErrors) (templ.Component, error) {
	var previews []*sqlc.MonitorResult
	if mon.Status == sqlc.MonitorStatusReady {
		res, err := a.monitor.ListMonitorResults(ctx, mon)
		if err != nil {
			return nil, err
		}
		previews = res
	}
	comp := appcomponents.MonitorDraftPage(appcomponents.MonitorDraftData{
		Monitor:          mon,
		Values:           values,
		ResultPreviews:   previews,
		ValidationErrors: validationErrs,
	})
	return comp, nil
}

func (a *app) monitorViewData(ctx context.Context, mon *sqlc.Monitor, u *sqlc.User) (appcomponents.MonitorViewData, error) {
	results, err := a.monitor.ListMonitorResults(ctx, mon)
	if err != nil {
		return appcomponents.MonitorViewData{}, err
	}

	nextScheduled, err := a.monitor.GetNextMonitorCheck(ctx, mon)
	if err != nil {
		return appcomponents.MonitorViewData{}, err
	}

	notifiers, err := a.monitor.ListMonitorNotifiers(ctx, mon)
	if err != nil {
		return appcomponents.MonitorViewData{}, err
	}

	activeIntegrations, err := a.userSettings.ActiveIntegrations(ctx, u.ID)
	if err != nil {
		return appcomponents.MonitorViewData{}, err
	}

	return appcomponents.MonitorViewData{
		Monitor:            mon,
		Results:            results,
		NextScheduledCheck: nextScheduled,
		Notifiers:          notifiers,
		ActiveIntegrations: activeIntegrations,
	}, nil
}

func (a *app) monitorComponent(ctx context.Context, mon *sqlc.Monitor, u *sqlc.User) (templ.Component, error) {
	data, err := a.monitorViewData(ctx, mon, u)
	if err != nil {
		return nil, err
	}
	comp := appcomponents.MonitorViewPage(data)
	return comp, nil
}

func (a *app) monitorNofifiersComponent(ctx context.Context, mon *sqlc.Monitor, u *sqlc.User) (templ.Component, error) {
	data, err := a.monitorViewData(ctx, mon, u)
	if err != nil {
		return nil, err
	}
	comp := appcomponents.MonitorNotifiers(data)
	return comp, nil
}

func (a *app) monitorViewGet(w http.ResponseWriter, r *http.Request, u *sqlc.User) {
	mon := a.monitorFromPath(w, r, u)
	if mon == nil {
		return
	}

	var err error
	var comp templ.Component

	if mon.Status != sqlc.MonitorStatusActive {
		comp, err = a.renderMonitorDraft(r.Context(), mon, monitor.NewUpdateMonitorDraftParams(mon), nil)
	} else {
		comp, err = a.monitorComponent(r.Context(), mon, u)
	}

	if a.internalServerError(err, w) {
		a.logger.Error("error rendering monitor view", "error", err)
		return
	}

	comp.Render(r.Context(), w)
}

func (a *app) monitorNewGet(w http.ResponseWriter, r *http.Request, _ *sqlc.User) {
	a.renderMonitorDraftNew(appcomponents.MonitorNewData{}, r, w)
}

func (a *app) renderMonitorDraftNew(data appcomponents.MonitorNewData, r *http.Request, w http.ResponseWriter) {
	appcomponents.MonitorNewPage(data).Render(r.Context(), w)
}

func (a *app) monitorUpdatePost(w http.ResponseWriter, r *http.Request, u *sqlc.User) {
	mon := a.monitorFromPath(w, r, u)
	if mon == nil {
		return
	}

	if err := r.ParseForm(); a.internalServerError(err, w) {
		a.logger.Error("error parsing monitor draft update form", "error", err)
		return
	}

	monitorDraftParams := monitor.UpdateMonitorDraftParams{
		CommonParams: monitor.CommonParams{
			Subject:      r.FormValue("Subject"),
			Instructions: r.FormValue("Instructions"),
		},
	}

	updatedMon, err := a.monitor.UpdateMonitorDraft(r.Context(), u.ID, mon.ID, monitorDraftParams)
	if err != nil {
		if validationErrs := validation.MapValidationErrors(err); validationErrs != nil {
			a.logger.Warn("failed validation when updating monitor", "validation_errors", validationErrs)
			comp, err := a.renderMonitorDraft(r.Context(), mon, monitorDraftParams, validationErrs)
			if a.internalServerError(err, w) {
				a.logger.Error("error rendering monitor draft after validation error", "error", err)
				return
			}
			comp.Render(r.Context(), w)
			return
		}
		a.internalServerError(err, w)
		a.logger.Error("error updating monitor", "error", err)
		return
	}

	comp, err := a.renderMonitorDraft(r.Context(), updatedMon, monitor.NewUpdateMonitorDraftParams(updatedMon), nil)
	if a.internalServerError(err, w) {
		a.logger.Error("error rendering monitor draft after update", "error", err)
		return
	}

	comp.Render(r.Context(), w)
}

func (a *app) monitorActivatePost(w http.ResponseWriter, r *http.Request, u *sqlc.User) {
	sse := datastar.NewSSE(w, r)

	monitorID := a.monitorIDFromPath(r)
	if monitorID == 0 {
		a.badRequest(fmt.Errorf("missing monitor id from path"), w)
		return
	}

	activatedMonitor, err := a.monitor.ActivateMonitorFromPreview(r.Context(), u.ID, monitorID)
	if a.internalServerError(err, w) {
		a.logger.Error("error activating monitor from preview", "error", err)
		return
	}

	sse.Redirectf("/app/monitors/%d", activatedMonitor.ID)
}

func (a *app) monitorCreatePost(w http.ResponseWriter, r *http.Request, u *sqlc.User) {
	if err := r.ParseForm(); a.internalServerError(err, w) {
		a.logger.Error("error parsing monitor draft create form", "error", err)
		return
	}

	newMonitor := monitor.CreateMonitorParams{
		UserID: u.ID,
		CommonParams: monitor.CommonParams{
			Subject: r.FormValue("Subject"),
		},
	}

	createdMonitor, err := a.monitor.CreateMonitor(r.Context(), newMonitor)
	if err != nil {
		if validationErrs := validation.MapValidationErrors(err); validationErrs != nil {
			a.logger.Info("failed validation when creating monitor", "validation_errors", validationErrs)
			a.renderMonitorDraftNew(appcomponents.MonitorNewData{
				ValidationErrors: validationErrs,
				Values:           newMonitor,
			}, r, w)
			return
		}
		a.internalServerError(err, w)
		a.logger.Error("error creating monitor", "error", err)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/app/monitors/%d", createdMonitor.ID), http.StatusSeeOther)
}

func (a *app) monitorCheckPost(w http.ResponseWriter, r *http.Request, u *sqlc.User) {
	sse := datastar.NewSSE(w, r)

	mon := a.monitorFromPath(w, r, u)
	if mon == nil {
		return
	}

	_, err := a.monitor.ScheduleMonitorCheck(r.Context(), mon, time.Now())
	if a.internalServerError(err, w) {
		a.logger.Error("error scheduling next check", "error", err)
		return
	}

	comp, err := a.monitorComponent(r.Context(), mon, u)
	if a.internalServerError(err, w) {
		a.logger.Error("error rendering monitor after scheduling check", "error", err)
		return
	}

	sse.PatchElementTempl(comp)
}

func (a *app) monitorDelete(w http.ResponseWriter, r *http.Request, user *sqlc.User) {
	sse := datastar.NewSSE(w, r)

	monitorID := a.monitorIDFromPath(r)
	err := a.monitor.DeleteMonitor(r.Context(), user.ID, monitorID)
	if a.internalServerError(err, w) {
		a.logger.Error("error deleting monitor", "error", err)
		return
	}

	sse.Redirect("/app/monitors")
}

func (a *app) monitorNotifierPost(w http.ResponseWriter, r *http.Request, u *sqlc.User) {
	mon := a.monitorFromPath(w, r, u)
	if mon == nil {
		return
	}

	notifierType := sqlc.Notifier(r.PathValue("type"))

	sse := datastar.NewSSE(w, r)

	_, err := a.monitor.CreateMonitorNotifier(r.Context(), mon, sqlc.Notifier(notifierType))
	if a.internalServerError(err, w) {
		a.logger.Error("error creating monitor notifier", "error", err)
		return
	}

	comp, err := a.monitorNofifiersComponent(r.Context(), mon, u)
	if a.internalServerError(err, w) {
		a.logger.Error("error rendering monitor notifiers after creating notifier", "error", err)
		return
	}

	sse.PatchElementTempl(comp)
}

func (a *app) monitorNotifierDelete(w http.ResponseWriter, r *http.Request, u *sqlc.User) {
	mon := a.monitorFromPath(w, r, u)
	if mon == nil {
		return
	}

	sse := datastar.NewSSE(w, r)

	notifierType := sqlc.Notifier(r.PathValue("type"))
	err := a.monitor.DeleteMonitorNotifier(r.Context(), mon, notifierType)
	if a.internalServerError(err, w) {
		a.logger.Error("error deleting monitor notifier", "error", err)
		return
	}

	comp, err := a.monitorNofifiersComponent(r.Context(), mon, u)
	if a.internalServerError(err, w) {
		a.logger.Error("error rendering monitor notifiers after creating notifier", "error", err)
		return
	}

	sse.PatchElementTempl(comp)
}

func (a *app) monitorFromPath(w http.ResponseWriter, r *http.Request, u *sqlc.User) *sqlc.Monitor {
	monitorID := a.monitorIDFromPath(r)
	if monitorID == 0 {
		a.badRequest(fmt.Errorf("no monitor id in path"), w)
		return nil
	}

	mon, err := a.monitor.GetMonitor(r.Context(), u.ID, monitorID)
	if errors.Is(err, monitor.ErrMonitorNotFound) {
		a.notFound(w)
		return nil
	}
	if a.internalServerError(err, w) {
		a.logger.Error("error getting monitor", "error", err)
		return nil
	}

	return mon
}

func (a *app) monitorIDFromPath(r *http.Request) int64 {
	monitorID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		return 0
	}
	return monitorID
}
