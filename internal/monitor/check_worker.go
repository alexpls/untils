package monitor

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
)

type CheckArgs struct {
	UserID         int64 `json:"user_id"`
	MonitorCheckID int64 `json:"monitor_check_id"`
}

func (CheckArgs) Kind() string {
	return "check"
}

type CheckWorker struct {
	river.WorkerDefaults[CheckArgs]
	service *Service
	logger  *slog.Logger
}

func NewCheckWorker(monitorService *Service, logger *slog.Logger) *CheckWorker {
	return &CheckWorker{
		service: monitorService,
		logger:  logger,
	}
}

func (w *CheckWorker) Work(ctx context.Context, job *river.Job[CheckArgs]) error {
	logger := w.logger.With("monitor_check_id", job.Args.MonitorCheckID)

	logger.Info("starting check worker")

	check, err := w.service.GetMonitorCheck(ctx, job.Args.MonitorCheckID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return river.JobCancel(fmt.Errorf("monitor check no longer exists"))
		}

		logger.Error("failed to get monitor check", "error", err)
		return err
	}

	if err = w.service.PerformMonitorCheck(ctx, job.Args.UserID, check); err != nil {
		logger.Error("failed to perform monitor check", "error", err)
		return err
	}

	return nil
}

func (w *CheckWorker) Timeout(job *river.Job[CheckArgs]) time.Duration {
	return 5 * time.Minute
}
