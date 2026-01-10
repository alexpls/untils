package main

import (
	"net/http"

	appcomponents "github.com/alexpls/untils/internal/components/app"
	"github.com/alexpls/untils/internal/db/sqlc"
	"github.com/starfederation/datastar-go/datastar"
)

func (a *app) dashboardGet(w http.ResponseWriter, r *http.Request, u *sqlc.User) {
	data := appcomponents.DashboardViewData{
		MonitorActivity: appcomponents.MonitorActivityWidgetData{
			Loading: appcomponents.LoadingStatusLoading,
		},
		CheckStats: appcomponents.CheckStatsWidgetData{
			Loading: appcomponents.LoadingStatusLoading,
		},
	}
	appcomponents.DashboardPage(data).Render(r.Context(), w)
}

func (a *app) dashboardEvents(w http.ResponseWriter, r *http.Request, u *sqlc.User) {
	sse := datastar.NewSSE(w, r)

	activity, err := a.monitor.ListMonitorActivity(r.Context(), u.ID)
	if a.internalServerError(err, w) {
		a.logger.Error("error listing monitor activity", "error", err)
		return
	}

	checkStats, err := a.monitor.GetMonitorCheckStats(r.Context(), u.ID)
	if a.internalServerError(err, w) {
		a.logger.Error("error listing monitor activity", "error", err)
		return
	}

	data := appcomponents.DashboardViewData{
		MonitorActivity: appcomponents.MonitorActivityWidgetData{
			Loading: appcomponents.LoadingStatusLoaded,
			Items:   activity,
		},
		CheckStats: appcomponents.CheckStatsWidgetData{
			Loading:    appcomponents.LoadingStatusLoaded,
			CheckStats: checkStats,
		},
	}

	comp := appcomponents.DashboardView(data)

	if err := sse.PatchElementTempl(comp); err != nil {
		a.logger.Error("error patching element", "error", err)
	}
}
