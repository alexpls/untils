package main

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	appcomponents "github.com/alexpls/untils_go/internal/components/app"
	"github.com/alexpls/untils_go/internal/db/sqlc"
	"github.com/alexpls/untils_go/internal/monitor"
	"github.com/alexpls/untils_go/internal/validation"
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

func (a *app) renderMonitorDraft(mon *sqlc.Monitor, values monitor.UpdateMonitorDraftParams, validationErrs validation.ValidationErrors, w http.ResponseWriter, r *http.Request) {
	var previews []*sqlc.MonitorResult
	if mon.Status == sqlc.MonitorStatusReady {
		res, err := a.monitor.ListMonitorResults(r.Context(), mon)
		if a.internalServerError(err, w) {
			a.logger.Error("error listing monitor check previews", "error", err)
			return
		}
		previews = res
	}
	appcomponents.MonitorDraftPage(appcomponents.MonitorDraftData{
		Monitor:          mon,
		Values:           values,
		ResultPreviews:   previews,
		ValidationErrors: validationErrs,
	}).Render(r.Context(), w)
}

func (a *app) renderMonitor(mon *sqlc.Monitor, u *sqlc.User, w http.ResponseWriter, r *http.Request) {
	results, err := a.monitor.ListMonitorResults(r.Context(), mon)
	if a.internalServerError(err, w) {
		a.logger.Error("error listing monitor check results", "error", err)
		return
	}

	nextScheduled, err := a.monitor.GetNextMonitorCheck(r.Context(), mon)
	if a.internalServerError(err, w) {
		a.logger.Error("error getting next monitor check", "error", err)
		return
	}

	notifiers, err := a.monitor.ListMonitorNotifiers(r.Context(), mon)
	if a.internalServerError(err, w) {
		a.logger.Error("error listing monitor notifiers", "error", err)
		return
	}

	activeIntegrations, err := a.userSettings.ActiveIntegrations(r.Context(), u.ID)
	if a.internalServerError(err, w) {
		a.logger.Error("error getting active integrations", "error", err)
		return
	}

	appcomponents.MonitorViewPage(appcomponents.MonitorViewData{
		Monitor:            mon,
		Results:            results,
		NextScheduledCheck: nextScheduled,
		Notifiers:          notifiers,
		ActiveIntegrations: activeIntegrations,
	}).Render(r.Context(), w)
}

func (a *app) monitorViewGet(w http.ResponseWriter, r *http.Request, u *sqlc.User) {
	mon := a.monitorFromPath(w, r, u)
	if mon == nil {
		return
	}

	if mon.Status != sqlc.MonitorStatusActive {
		a.renderMonitorDraft(mon, monitor.NewUpdateMonitorDraftParams(mon), nil, w, r)
	} else {
		a.renderMonitor(mon, u, w, r)
	}
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
			a.renderMonitorDraft(mon, monitorDraftParams, validationErrs, w, r)
			return
		}
		a.internalServerError(err, w)
		a.logger.Error("error updating monitor", "error", err)
		return
	}

	a.renderMonitorDraft(updatedMon, monitor.NewUpdateMonitorDraftParams(updatedMon), nil, w, r)
}

func (a *app) monitorActivatePost(w http.ResponseWriter, r *http.Request, u *sqlc.User) {
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

	http.Redirect(w, r, fmt.Sprintf("/app/monitors/%d", activatedMonitor.ID), http.StatusSeeOther)
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
	mon := a.monitorFromPath(w, r, u)
	if mon == nil {
		return
	}

	_, err := a.monitor.ScheduleMonitorCheck(r.Context(), mon, time.Now())
	if a.internalServerError(err, w) {
		a.logger.Error("error scheduling next check", "error", err)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/app/monitors/%d", mon.ID), http.StatusSeeOther)
}

func (a *app) monitorDelete(w http.ResponseWriter, r *http.Request, user *sqlc.User) {
	monitorID := a.monitorIDFromPath(r)
	err := a.monitor.DeleteMonitor(r.Context(), user.ID, monitorID)
	if a.internalServerError(err, w) {
		a.logger.Error("error deleting monitor", "error", err)
		return
	}
	http.Redirect(w, r, "/app/monitors", http.StatusSeeOther)
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

func (a *app) monitorNotifierPost(w http.ResponseWriter, r *http.Request, u *sqlc.User) {
	mon := a.monitorFromPath(w, r, u)
	if mon == nil {
		return
	}

	if err := r.ParseForm(); a.internalServerError(err, w) {
		a.logger.Error("error parsing notifier form", "error", err)
		return
	}

	notifierType := sqlc.Notifier(r.FormValue("Type"))
	_, err := a.monitor.CreateMonitorNotifier(r.Context(), mon, notifierType)
	if a.internalServerError(err, w) {
		a.logger.Error("error creating monitor notifier", "error", err)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/app/monitors/%d", mon.ID), http.StatusSeeOther)
}

func (a *app) monitorNotifierDelete(w http.ResponseWriter, r *http.Request, u *sqlc.User) {
	mon := a.monitorFromPath(w, r, u)
	if mon == nil {
		return
	}

	notifierType := sqlc.Notifier(r.PathValue("type"))
	err := a.monitor.DeleteMonitorNotifier(r.Context(), mon, notifierType)
	if a.internalServerError(err, w) {
		a.logger.Error("error deleting monitor notifier", "error", err)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/app/monitors/%d", mon.ID), http.StatusSeeOther)
}
