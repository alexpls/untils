package monitor

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/alexpls/untils/internal/errortypes"
	"github.com/alexpls/untils/internal/types"
	"github.com/riverqueue/river"
)

type ValidateMonitorArgs struct {
	UserID    int64 `json:"user_id"`
	MonitorID int64 `json:"monitor_id"`
}

func (ValidateMonitorArgs) Kind() string {
	return "validate_draft"
}

func (ValidateMonitorArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue: types.RiverBrowserQueue,
	}
}

type ValidateMonitorWorker struct {
	river.WorkerDefaults[ValidateMonitorArgs]
	service *Service
	logger  *slog.Logger
}

func NewValidateMonitorWorker(monitorService *Service, logger *slog.Logger) *ValidateMonitorWorker {
	return &ValidateMonitorWorker{
		service: monitorService,
		logger:  logger,
	}
}

func (w *ValidateMonitorWorker) Timeout(job *river.Job[ValidateMonitorArgs]) time.Duration {
	return 3 * time.Minute
}

func (w *ValidateMonitorWorker) Work(ctx context.Context, job *river.Job[ValidateMonitorArgs]) error {
	logger := w.logger.With("monitor_id", job.Args.MonitorID)

	logger.InfoContext(ctx, "starting validate monitor worker")

	mon, err := w.service.GetMonitor(ctx, job.Args.UserID, job.Args.MonitorID)
	if err != nil {
		if errors.Is(err, &errortypes.ResourceNotFoundError{}) {
			return river.JobCancel(fmt.Errorf("monitor no longer exists"))
		}
		logger.ErrorContext(ctx, "failed to get monitor", "error", err)
		return err
	}

	if err = w.service.ValidateMonitor(ctx, mon); err != nil {
		if er, found := errors.AsType[*errortypes.InvalidMonitorStatusTransitionError](err); found {
			return river.JobCancel(er)
		}
		if isStaleMonitorWorkError(err) {
			return river.JobCancel(err)
		}
		logger.ErrorContext(ctx, "failed to validate monitor", "error", err)
		return err
	}

	return nil
}
