package monitor

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/riverqueue/river"
)

type ValidateMonitorArgs struct {
	UserID    int64 `json:"user_id"`
	MonitorID int64 `json:"monitor_id"`
}

func (ValidateMonitorArgs) Kind() string {
	return "validate_draft"
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

func (w *ValidateMonitorWorker) Work(ctx context.Context, job *river.Job[ValidateMonitorArgs]) error {
	logger := w.logger.With("monitor_id", job.Args.MonitorID)
	start := time.Now()

	logger.Info("starting validate monitor worker")

	mon, err := w.service.GetMonitor(ctx, job.Args.UserID, job.Args.MonitorID)
	if err != nil {
		if errors.Is(err, ErrMonitorNotFound) {
			return river.JobCancel(fmt.Errorf("monitor no longer exists"))
		}
		logger.Error("failed to get monitor", "error", err)
		return err
	}

	if err = w.service.ValidateMonitor(ctx, mon); err != nil {
		var er *ErrInvalidStatusTransition
		if errors.As(err, &er) {
			return river.JobCancel(er)
		}
		logger.Error("failed to validate monitor", "error", err)
		return err
	}

	logger.Info("successfully validated monitor", "duration_ms", time.Since(start).Milliseconds())

	return nil
}

func (w *ValidateMonitorWorker) Timeout(job *river.Job[ValidateMonitorArgs]) time.Duration {
	return 3 * time.Minute
}
