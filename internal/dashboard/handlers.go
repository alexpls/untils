package dashboard

import (
	"log/slog"
	"net/http"

	"github.com/alexpls/untils/internal/models"
	"github.com/alexpls/untils/internal/monitor"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/starfederation/datastar-go/datastar"
)

// Handlers contains the HTTP handlers for dashboard routes
type Handlers struct {
	queries       *models.Queries
	pool          *pgxpool.Pool
	monitorEvents *monitor.DBEventHandler
	logger        *slog.Logger
}

// NewHandlers creates a new Handlers instance
func NewHandlers(queries *models.Queries, pool *pgxpool.Pool, monitorEvents *monitor.DBEventHandler, logger *slog.Logger) *Handlers {
	return &Handlers{
		queries:       queries,
		pool:          pool,
		monitorEvents: monitorEvents,
		logger:        logger,
	}
}

// Get handles GET /app
func (h *Handlers) Get(w http.ResponseWriter, r *http.Request, user *models.User) {
	data := DashboardViewData{
		MonitorActivity: MonitorActivityWidgetData{
			Loading: LoadingStatusLoading,
		},
		CheckStats: CheckStatsWidgetData{
			Loading: LoadingStatusLoading,
		},
	}
	if err := DashboardPage(data).Render(r.Context(), w); err != nil {
		h.logger.Error("error rendering dashboard page", "error", err)
	}
}

// Events handles GET /app/dashboard/events (SSE)
func (h *Handlers) Events(w http.ResponseWriter, r *http.Request, user *models.User) {
	sse := datastar.NewSSE(w, r)

	ch := h.monitorEvents.SubscribeUser(r.Context(), user.ID)

	// only use view transitions for the initial load, otherwise if they
	// happen on every change to monitors it can be distracting and potentially
	// fire for no-ops.
	useViewTransition := true

	for {
		activity, err := h.queries.ListMonitorActivity(r.Context(), h.pool, user.ID)
		if err != nil {
			h.logger.Error("error listing monitor activity", "error", err)
			return
		}

		checkStats, err := h.queries.GetMonitorCheckStats(r.Context(), h.pool, user.ID)
		if err != nil {
			h.logger.Error("error getting monitor check stats", "error", err)
			return
		}

		dailyCheckCounts, err := h.queries.GetDailyMonitorCheckCounts(r.Context(), h.pool, user.ID)
		if err != nil {
			h.logger.Error("error getting daily monitor check counts", "error", err)
			return
		}

		data := DashboardViewData{
			MonitorActivity: MonitorActivityWidgetData{
				Loading: LoadingStatusLoaded,
				Items:   activity,
			},
			CheckStats: CheckStatsWidgetData{
				Loading:          LoadingStatusLoaded,
				CheckStats:       checkStats,
				DailyCheckCounts: dailyCheckCounts,
			},
		}

		comp := DashboardView(data)

		var viewTransitionOpt datastar.PatchElementOption
		if useViewTransition {
			viewTransitionOpt = datastar.WithViewTransitions()
		} else {
			viewTransitionOpt = datastar.WithoutViewTransitions()
		}

		if err := sse.PatchElementTempl(comp, viewTransitionOpt); err != nil {
			h.logger.Error("error patching element", "error", err)
		}

		select {
		case <-ch:
			useViewTransition = false
			continue
		case <-sse.Context().Done():
			return
		}
	}
}
