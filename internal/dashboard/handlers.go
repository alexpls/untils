package dashboard

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/a-h/templ"
	"github.com/alexpls/untils/internal/db"
	"github.com/alexpls/untils/internal/models"
	"github.com/alexpls/untils/internal/monitor"
)

// Handlers contains the HTTP handlers for dashboard routes
type Handlers struct {
	queries       *models.Queries
	db            db.DB
	monitorEvents *monitor.DBEventHandler
	logger        *slog.Logger
}

// NewHandlers creates a new Handlers instance
func NewHandlers(queries *models.Queries, db db.DB, monitorEvents *monitor.DBEventHandler, logger *slog.Logger) *Handlers {
	return &Handlers{
		queries:       queries,
		db:            db,
		monitorEvents: monitorEvents,
		logger:        logger,
	}
}

// ViewDashboard handles GET /app
func (h *Handlers) ViewDashboard(w http.ResponseWriter, r *http.Request, user *models.User) {
	patcher := monitor.ConditionalPatchRenderer{
		Logger: h.logger,
		Renderer: func(patch bool) (templ.Component, error) {
			data, err := h.dashboardViewData(r.Context(), user.ID, LoadingStatusLoaded)
			if err != nil {
				return nil, err
			}
			if patch {
				return DashboardView(data), nil
			}
			return DashboardPage(data), nil
		},
		Updater: func(ctx context.Context) (<-chan struct{}, error) {
			return h.monitorEvents.SubscribeUser(ctx, user.ID), nil
		},
	}
	patcher.Handle(w, r)
}

func (h *Handlers) dashboardViewData(ctx context.Context, userID int64, loading LoadingStatus) (DashboardViewData, error) {
	if loading == LoadingStatusLoading {
		return DashboardViewData{
			MonitorActivity: MonitorActivityWidgetData{
				Loading: LoadingStatusLoading,
			},
			CheckStats: CheckStatsWidgetData{
				Loading: LoadingStatusLoading,
			},
		}, nil
	}

	activity, err := h.queries.ListMonitorActivity(ctx, h.db, userID)
	if err != nil {
		return DashboardViewData{}, err
	}

	checkStats, err := h.queries.GetMonitorCheckStats(ctx, h.db, userID)
	if err != nil {
		return DashboardViewData{}, err
	}

	dailyCheckCounts, err := h.queries.GetDailyMonitorCheckCounts(ctx, h.db, userID)
	if err != nil {
		return DashboardViewData{}, err
	}

	return DashboardViewData{
		MonitorActivity: MonitorActivityWidgetData{
			Loading: LoadingStatusLoaded,
			Items:   activity,
		},
		CheckStats: CheckStatsWidgetData{
			Loading:          LoadingStatusLoaded,
			CheckStats:       checkStats,
			DailyCheckCounts: dailyCheckCounts,
		},
	}, nil
}
