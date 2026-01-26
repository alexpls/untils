package monitor

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/alexpls/untils/internal/db"
	"github.com/alexpls/untils/internal/llm"
	"github.com/alexpls/untils/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/riverqueue/river"
)

var MonitorCheckTerminalStatuses = []models.MonitorCheckStatus{
	models.MonitorCheckStatusFailed,
	models.MonitorCheckStatusSkipped,
	models.MonitorCheckStatusSuccess,
}

func (s *Service) GetMonitorCheck(ctx context.Context, id int64) (*models.MonitorCheck, error) {
	check, err := s.queries.GetMonitorCheck(ctx, s.db, id)
	if err != nil {
		return nil, fmt.Errorf("getting monitor check: %w", err)
	}
	return check, nil
}

func (s *Service) GetNextMonitorCheck(ctx context.Context, monitor *models.Monitor) (*models.MonitorCheck, error) {
	check, err := s.queries.GetNextMonitorCheck(ctx, s.db, monitor.ID)
	if err != nil {
		return nil, fmt.Errorf("getting next monitor check: %w", err)
	}
	return check, nil
}

// GetInProgressMonitorCheck returns the in-progress monitor check for the given monitor,
// or nil if there is no in-progress check.
func (s *Service) GetInProgressMonitorCheck(ctx context.Context, monitor *models.Monitor) (*models.MonitorCheck, error) {
	check, err := s.queries.GetInProgressMonitorCheck(ctx, s.db, monitor.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("getting in progress monitor check: %w", err)
	}
	return check, nil
}

func (s *Service) ScheduleMonitorCheck(ctx context.Context, monitor *models.Monitor, scheduledFor time.Time) (*models.MonitorCheck, error) {
	return db.WithTxV(s.db, ctx, func(tx pgx.Tx) (*models.MonitorCheck, error) {
		check, err := s.scheduleMonitorCheckTx(ctx, tx, monitor, scheduledFor)
		if err != nil {
			return nil, fmt.Errorf("scheduling monitor check: %w", err)
		}
		return check, nil
	})
}

func (s *Service) scheduleMonitorCheckTx(ctx context.Context, tx pgx.Tx, monitor *models.Monitor, scheduledFor time.Time) (*models.MonitorCheck, error) {
	if err := s.queries.SkipPendingChecks(ctx, tx, monitor.ID); err != nil {
		return nil, fmt.Errorf("skipping pending checks: %w", err)
	}

	check, err := s.queries.CreateMonitorCheck(ctx, tx, &models.CreateMonitorCheckParams{
		MonitorID:    monitor.ID,
		Status:       models.MonitorCheckStatusScheduled,
		ScheduledFor: scheduledFor,
	})
	if err != nil {
		return nil, fmt.Errorf("creating monitor check: %w", err)
	}

	var opts river.InsertOpts
	if scheduledFor.After(time.Now()) {
		opts.ScheduledAt = scheduledFor
	}

	_, err = s.river.InsertTx(ctx, tx, CheckArgs{
		UserID:         monitor.UserID,
		MonitorCheckID: check.ID,
	}, &opts)
	if err != nil {
		return nil, fmt.Errorf("inserting river job: %w", err)
	}

	return check, nil
}

func (s *Service) PerformMonitorCheck(
	ctx context.Context,
	userID int64,
	check *models.MonitorCheck,
	scheduleNext bool,
	userFeedback string,
) error {
	if slices.Contains(MonitorCheckTerminalStatuses, check.Status) {
		s.logger.WarnContext(ctx, "tried to perform a monitor check that is already in a terminal status", "check_id", check.ID, "status", check.Status)
		return nil
	}

	monitor, err := s.GetMonitor(ctx, userID, check.MonitorID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			s.logger.WarnContext(ctx, "tried to perform a check on a non-existent monitor", "monitor_id", check.MonitorID)
			return nil
		}
		return fmt.Errorf("getting monitor: %w", err)
	}

	latest, err := db.WithTxV(s.db, ctx, func(tx pgx.Tx) ([]*models.GetPreviousResultsWithCheckRow, error) {
		latest, err := s.queries.GetPreviousResultsWithCheck(ctx, tx, monitor.ID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				latest = nil
			} else {
				return nil, fmt.Errorf("getting previous results: %w", err)
			}
		}

		if scheduleNext {
			_, err = s.scheduleMonitorCheckTx(ctx, tx, monitor, time.Now().Add(monitorCheckFrequency))
			if err != nil {
				return nil, err
			}
		}

		if err = s.queries.UpdateMonitorCheckChecking(ctx, tx, check.ID); err != nil {
			return nil, fmt.Errorf("updating monitor check status: %w", err)
		}

		return latest, nil
	})
	if err != nil {
		return err
	}

	checker := s.llm.NewCheckWorkflow()

	result, err := checker.Run(ctx, &llm.CheckParams{
		UserID:          userID,
		MonitorCheckID:  check.ID,
		Subject:         monitor.Subject.String,
		PreviousResults: latest,
	})
	if err != nil {
		if cerr := s.queries.UpdateMonitorCheckFailed(ctx, s.db, &models.UpdateMonitorCheckFailedParams{
			FailureReason: pgtype.Text{String: err.Error(), Valid: true},
			ID:            check.ID,
		}); cerr != nil {
			return fmt.Errorf("updating check status after llm error: %w", cerr)
		}
		return fmt.Errorf("prompting llm: %w", err)
	}

	err = db.WithTx(s.db, ctx, func(tx pgx.Tx) error {
		if err := s.validateMonitorsSameVersion(ctx, tx, monitor); err != nil {
			return err
		}

		if err := s.bumpMonitorVersion(ctx, tx, monitor); err != nil {
			return fmt.Errorf("bumping monitor version: %w", err)
		}

		if err := s.queries.UpdateMonitorCheckSuccess(ctx, tx, &models.UpdateMonitorCheckSuccessParams{
			ID:     check.ID,
			Result: result,
		}); err != nil {
			return fmt.Errorf("updating monitor check to success status: %w", err)
		}

		if result.DifferentToPrevious || len(latest) == 0 {
			params := CheckResultToCreateMonitorResultParams(check.MonitorID, check.ID, result)
			if _, err := s.queries.CreateMonitorResult(ctx, tx, params); err != nil {
				return fmt.Errorf("creating check result: %w", err)
			}
		} else {
			if err := s.queries.AppendConfirmingCheckIDToResult(ctx, tx, &models.AppendConfirmingCheckIDToResultParams{
				MonitorResultID:           latest[0].MonitorResult.ID,
				ConfirmingCheckIDToAppend: check.ID,
			}); err != nil {
				return fmt.Errorf("appending confirming check id to result: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	if result.DifferentToPrevious {
		lastResult := "(none)"
		if len(latest) > 0 {
			lastResult = latest[0].MonitorResult.Result
		}

		if err = s.SendNotifications(ctx, SendNotificationsParams{
			Monitor:   monitor,
			NewResult: result.ResultPlaintext,
			OldResult: lastResult,
		}); err != nil {
			return err
		}
	}

	return nil
}
