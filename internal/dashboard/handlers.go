package dashboard

import (
	"log/slog"
	"net/http"

	"github.com/alexpls/untils/internal/db/sqlc"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/starfederation/datastar-go/datastar"
)

// Handlers contains the HTTP handlers for dashboard routes
type Handlers struct {
	queries *sqlc.Queries
	pool    *pgxpool.Pool
	logger  *slog.Logger
}

// NewHandlers creates a new Handlers instance
func NewHandlers(queries *sqlc.Queries, pool *pgxpool.Pool, logger *slog.Logger) *Handlers {
	return &Handlers{
		queries: queries,
		pool:    pool,
		logger:  logger,
	}
}

// Get handles GET /app
func (h *Handlers) Get(w http.ResponseWriter, r *http.Request, user *sqlc.User) {
	data := DashboardViewData{
		MonitorActivity: MonitorActivityWidgetData{
			Loading: LoadingStatusLoading,
		},
		CheckStats: CheckStatsWidgetData{
			Loading: LoadingStatusLoading,
		},
	}
	DashboardPage(data).Render(r.Context(), w)
}

// Events handles GET /app/dashboard/events (SSE)
func (h *Handlers) Events(w http.ResponseWriter, r *http.Request, user *sqlc.User) {
	sse := datastar.NewSSE(w, r)

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

	if err := sse.PatchElementTempl(comp); err != nil {
		h.logger.Error("error patching element", "error", err)
	}
}
