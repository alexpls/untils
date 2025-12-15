package monitor

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/riverqueue/river"
)

type PreviewMonitorArgs struct {
	UserID    int64 `json:"user_id"`
	MonitorID int64 `json:"monitor_id"`
}

func (PreviewMonitorArgs) Kind() string {
	return "preview_monitor"
}

type PreviewMonitorWorker struct {
	river.WorkerDefaults[PreviewMonitorArgs]
	service *Service
	logger  *slog.Logger
}

func NewPreviewMonitorWorker(monitorService *Service, logger *slog.Logger) *PreviewMonitorWorker {
	return &PreviewMonitorWorker{
		service: monitorService,
		logger:  logger,
	}
}

func (w *PreviewMonitorWorker) Work(ctx context.Context, job *river.Job[PreviewMonitorArgs]) error {
	logger := w.logger.With("monitor_id", job.Args.MonitorID)
	start := time.Now()

	logger.Info("starting preview monitor worker")

	monitor, err := w.service.GetMonitor(ctx, job.Args.UserID, job.Args.MonitorID)
	if err != nil {
		if errors.Is(err, ErrMonitorNotFound) {
			return river.JobCancel(fmt.Errorf("monitor no longer exists"))
		}
		logger.Error("failed to get monitor", "error", err)
		return err
	}

	if err = w.service.PreviewMonitor(ctx, monitor); err != nil {
		logger.Error("failed to preview monitor", "error", err)
		return err
	}

	logger.Info("successfully previewed monitor", "duration_ms", time.Since(start).Milliseconds())

	return nil
}

func (w *PreviewMonitorWorker) Timeout(job *river.Job[PreviewMonitorArgs]) time.Duration {
	return 5 * time.Minute
}
