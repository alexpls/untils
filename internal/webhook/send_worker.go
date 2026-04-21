package webhook

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
)

const SendMaxAttempts = 10

type SendArgs struct {
	UserID          int64 `json:"user_id"`
	WebhookTargetID int64 `json:"webhook_target_id"`
	MonitorID       int64 `json:"monitor_id"`
	NewResultID     int64 `json:"new_result_id"`
	OldResultID     int64 `json:"old_result_id"`
}

func (SendArgs) Kind() string {
	return "webhook_send"
}

func (SendArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{MaxAttempts: SendMaxAttempts}
}

type SendWorker struct {
	river.WorkerDefaults[SendArgs]
	service *Service
	logger  *slog.Logger
}

func NewSendWorker(service *Service, logger *slog.Logger) *SendWorker {
	return &SendWorker{
		service: service,
		logger:  logger,
	}
}

func (w *SendWorker) Timeout(job *river.Job[SendArgs]) time.Duration {
	return 20 * time.Second
}

func (w *SendWorker) Work(ctx context.Context, job *river.Job[SendArgs]) error {
	logger := w.logger.With(
		"user_id", job.Args.UserID,
		"webhook_target_id", job.Args.WebhookTargetID,
		"monitor_id", job.Args.MonitorID,
		"new_result_id", job.Args.NewResultID,
		"old_result_id", job.Args.OldResultID,
		"attempt", job.Attempt,
		"max_attempts", job.MaxAttempts,
	)

	_, err := w.service.SendMonitorNewResult(ctx, job.Args)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			logger.WarnContext(ctx, "webhook send canceled because target, monitor, or result no longer exists", "error", err)
			return river.JobCancel(fmt.Errorf("webhook target, monitor, or result no longer exists: %w", err))
		}

		logger.ErrorContext(ctx, "webhook send failed", "error", err)
		return err
	}

	return nil
}
