package monitor

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/alexpls/untils_go/internal/db"
	"github.com/alexpls/untils_go/internal/db/sqlc"
	"github.com/alexpls/untils_go/internal/llm"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/riverqueue/river"
)

var MonitorCheckTerminalStatuses = []sqlc.MonitorCheckStatus{
	sqlc.MonitorCheckStatusFailed,
	sqlc.MonitorCheckStatusSkipped,
	sqlc.MonitorCheckStatusSuccess,
}

func (s *Service) GetMonitorCheck(ctx context.Context, id int64) (*sqlc.MonitorCheck, error) {
	check, err := s.queries.GetMonitorCheck(ctx, s.pool, id)
	if err != nil {
		return nil, fmt.Errorf("getting monitor check: %w", err)
	}
	return check, nil
}

func (s *Service) GetNextMonitorCheck(ctx context.Context, monitor *sqlc.Monitor) (*sqlc.MonitorCheck, error) {
	check, err := s.queries.GetNextMonitorCheck(ctx, s.pool, monitor.ID)
	if err != nil {
		return nil, fmt.Errorf("getting next monitor check: %w", err)
	}
	return check, nil
}

func (s *Service) ScheduleMonitorCheck(ctx context.Context, monitor *sqlc.Monitor, scheduledFor time.Time) (*sqlc.MonitorCheck, error) {
	return db.WithTxV(s.pool, ctx, func(tx pgx.Tx) (*sqlc.MonitorCheck, error) {
		check, err := s.scheduleMonitorCheckTx(ctx, tx, monitor, scheduledFor)
		if err != nil {
			return nil, fmt.Errorf("scheduling monitor check: %w", err)
		}
		return check, nil
	})
}

func (s *Service) scheduleMonitorCheckTx(ctx context.Context, tx pgx.Tx, monitor *sqlc.Monitor, scheduledFor time.Time) (*sqlc.MonitorCheck, error) {
	if err := s.queries.SkipPendingChecks(ctx, tx, monitor.ID); err != nil {
		return nil, fmt.Errorf("skipping pending checks: %w", err)
	}

	check, err := s.queries.CreateMonitorCheck(ctx, tx, &sqlc.CreateMonitorCheckParams{
		MonitorID:    monitor.ID,
		Status:       sqlc.MonitorCheckStatusScheduled,
		ScheduledFor: scheduledFor,
	})
	if err != nil {
		return nil, fmt.Errorf("creating monitor check: %w", err)
	}

	_, err = s.river.InsertTx(ctx, tx, CheckArgs{
		UserID:         monitor.UserID,
		MonitorCheckID: check.ID,
	}, &river.InsertOpts{
		ScheduledAt: scheduledFor,
	})
	if err != nil {
		return nil, fmt.Errorf("inserting river job: %w", err)
	}

	return check, nil
}

func (s *Service) PerformMonitorCheck(ctx context.Context, userID int64, check *sqlc.MonitorCheck) error {
	if slices.Contains(MonitorCheckTerminalStatuses, check.Status) {
		return nil
	}

	monitor, err := s.GetMonitor(ctx, userID, check.MonitorID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			s.logger.Warn("tried to perform a check on a non-existent monitor", "monitor_id", check.MonitorID)
			return nil
		}
		return fmt.Errorf("getting monitor: %w", err)
	}

	latest, err := db.WithTxV(s.pool, ctx, func(tx pgx.Tx) ([]*sqlc.MonitorResult, error) {
		latest, err := s.queries.GetLatestMonitorResults(ctx, tx, monitor.ID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				latest = nil
			} else {
				return nil, fmt.Errorf("getting latest check result: %w", err)
			}
		}

		_, err = s.scheduleMonitorCheckTx(ctx, tx, monitor, time.Now().Add(monitorCheckFrequency))
		if err != nil {
			return nil, err
		}

		if err = s.queries.UpdateMonitorCheckChecking(ctx, tx, check.ID); err != nil {
			return nil, fmt.Errorf("updating monitor check status: %w", err)
		}

		return latest, nil
	})
	if err != nil {
		return err
	}

	prevResults := make([]llm.PreviousResult, len(latest))
	for i, r := range latest {
		prevResults[i] = llm.PreviousResult{
			DateChecked:       r.CreatedAt,
			ResponsePlaintext: r.Result,
		}
	}

	expert := llm.NewExpert(monitor.Expert.String, s.llm)
	result, err := expert.PerformCheck(ctx, &llm.CheckParams{
		Subject:         monitor.Subject.String,
		PreviousResults: prevResults,
		Instructions:    monitor.Instructions.String,
	})
	if err != nil {
		if cerr := s.queries.UpdateMonitorCheckFailed(ctx, s.pool, &sqlc.UpdateMonitorCheckFailedParams{
			FailureReason: pgtype.Text{String: err.Error(), Valid: true},
			ID:            check.ID,
		}); cerr != nil {
			return fmt.Errorf("updating check status after llm error: %w", cerr)
		}
		return fmt.Errorf("prompting llm: %w", err)
	}

	err = db.WithTx(s.pool, ctx, func(tx pgx.Tx) error {
		if err := s.validateMonitorsSameVersion(ctx, tx, monitor); err != nil {
			return err
		}

		if err := s.bumpMonitorVersion(ctx, tx, monitor); err != nil {
			return fmt.Errorf("bumping monitor version: %w", err)
		}

		if err := s.queries.UpdateMonitorCheckSuccess(ctx, tx, check.ID); err != nil {
			return fmt.Errorf("updating monitor check to success status: %w", err)
		}

		if result.DifferentToPrevious || len(latest) == 0 {
			params := checkResponseToCreateMonitorResultParams(check.MonitorID, check.ID, result)
			if _, err := s.queries.CreateMonitorResult(ctx, tx, params); err != nil {
				return fmt.Errorf("creating check result: %w", err)
			}
		} else {
			if err := s.queries.AppendConfirmingCheckIDToResult(ctx, tx, &sqlc.AppendConfirmingCheckIDToResultParams{
				MonitorResultID:           latest[0].ID,
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
			lastResult = latest[0].Result
		}

		notifMessage := fmt.Sprintf("Change detected: %s (was %s)", result.ResponsePlaintext, lastResult)
		if err = s.SendNotifications(ctx, SendNotificationsParams{
			Monitor: monitor,
			Message: notifMessage,
		}); err != nil {
			return err
		}
	}

	return nil
}
