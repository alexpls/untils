package monitor

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alexpls/untils/internal/db"
	"github.com/alexpls/untils/internal/db/sqlc"
	"github.com/alexpls/untils/internal/llm"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var monitorCheckFrequency = 6 * time.Hour

func (s *Service) ListMonitors(ctx context.Context, userID int64) ([]*sqlc.Monitor, error) {
	return s.queries.ListMonitors(ctx, s.pool, userID)
}

func (s *Service) GetMonitor(ctx context.Context, userID, monitorID int64) (*sqlc.Monitor, error) {
	monitor, err := s.queries.GetMonitor(ctx, s.pool, &sqlc.GetMonitorParams{
		UserID: userID,
		ID:     monitorID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrMonitorNotFound
		}
		return nil, fmt.Errorf("getting monitor: %w", err)
	}

	return monitor, nil
}

type CommonParams struct {
	Subject      string `validate:"required,min=10,max=5000"`
	Instructions string `validate:"omitempty,min=10,max=10000"`
}

type CreateMonitorParams struct {
	UserID int64 `validate:"required"`
	CommonParams
}

func (s *Service) CreateMonitor(ctx context.Context, params CreateMonitorParams) (*sqlc.Monitor, error) {
	if err := s.validate.Struct(params); err != nil {
		return nil, err
	}

	return db.WithTxV(s.pool, ctx, func(tx pgx.Tx) (*sqlc.Monitor, error) {
		created, err := s.queries.CreateMonitor(ctx, tx, &sqlc.CreateMonitorParams{
			UserID:       params.UserID,
			Subject:      pgtype.Text{String: params.Subject, Valid: true},
			Instructions: pgtype.Text{String: "", Valid: true},
		})
		if err != nil {
			return nil, fmt.Errorf("creating monitor: %w", err)
		}

		if _, err = s.river.InsertTx(ctx, tx, ValidateMonitorArgs{
			UserID:    created.UserID,
			MonitorID: created.ID,
		}, nil); err != nil {
			return nil, fmt.Errorf("enqueuing validate monitor job: %w", err)
		}

		return created, nil
	})
}

func (s *Service) DeleteMonitor(ctx context.Context, userID, monitorID int64) error {
	err := s.queries.DeleteMonitor(ctx, s.pool, &sqlc.DeleteMonitorParams{
		UserID:    userID,
		MonitorID: monitorID,
	})
	if err != nil {
		return fmt.Errorf("deleting monitor: %w", err)
	}
	return nil
}

// TODO: the "validate" name should be more like triage now to align with package llm
func (s *Service) ValidateMonitor(ctx context.Context, monitor *sqlc.Monitor) error {
	if monitor.Status != sqlc.MonitorStatusValidating {
		return fmt.Errorf("monitor: must be in 'validating' status, got: %s", monitor.Status)
	}

	if err := db.WithTx(s.pool, ctx, func(tx pgx.Tx) error {
		return s.deleteMonitorRelations(ctx, tx, monitor.ID)
	}); err != nil {
		return err
	}

	triage := llm.NewTriageWorkflow(s.llm)

	trigageRes, err := triage.Run(ctx, &llm.TriageParams{
		Subject:      monitor.Subject.String,
		Instructions: monitor.Instructions.String,
	})
	if err != nil {
		return fmt.Errorf("triage workflow: %w", err)
	}

	if !trigageRes.Approved {
		if err = s.queries.RejectMonitor(ctx, s.pool, &sqlc.RejectMonitorParams{
			ID:             monitor.ID,
			UserID:         monitor.UserID,
			RejectedReason: pgtype.Text{String: trigageRes.RejectedReason, Valid: true},
		}); err != nil {
			return fmt.Errorf("rejecting monitor: %w", err)
		}
		return nil
	}

	var check *sqlc.MonitorCheck

	if err := db.WithTx(s.pool, ctx, func(tx pgx.Tx) error {
		if err := s.validateMonitorsSameVersion(ctx, tx, monitor); err != nil {
			return err
		}

		monitor, err = s.queries.UpdateMonitorStatus(ctx, tx, &sqlc.UpdateMonitorStatusParams{
			ID:     monitor.ID,
			UserID: monitor.UserID,
			Status: sqlc.MonitorStatusPreviewing,
		})
		if err != nil {
			return err
		}

		check, err = s.queries.CreateMonitorCheck(ctx, tx, &sqlc.CreateMonitorCheckParams{
			MonitorID:    monitor.ID,
			Status:       sqlc.MonitorCheckStatusScheduled,
			ScheduledFor: time.Now(),
		})
		if err != nil {
			return err
		}

		return nil
	}); err != nil {
		return err
	}

	// TODO: go back to the triager if the check fails
	err = s.PerformMonitorCheck(ctx, monitor.UserID, check, false)
	if err != nil {
		return fmt.Errorf("performing monitor check: %w", err)
	}

	_, err = s.queries.UpdateMonitorToReady(ctx, s.pool, &sqlc.UpdateMonitorToReadyParams{
		MonitorID: monitor.ID,
		UserID:    monitor.UserID,
		Subject:   monitor.Subject,
		Expert:    pgtype.Text{String: "default", Valid: true},
	})

	return err
}

func CheckResultToCreateMonitorResultParams(monitorID, checkID int64, res *sqlc.CheckResult) *sqlc.CreateMonitorResultParams {
	resultDate := pgtype.Timestamptz{Time: time.Time{}, Valid: false}
	resultDatePastTenseVerb := pgtype.Text{String: "", Valid: false}

	if res.Date.Date != "" {
		parsed, err := time.Parse("2006-01-02", res.Date.Date)
		if err == nil {
			resultDate.Time = parsed
			resultDate.Valid = true

			if res.Date.PastTenseVerb != "" {
				resultDatePastTenseVerb.String = res.Date.PastTenseVerb
				resultDatePastTenseVerb.Valid = true
			}
		}
	}

	return &sqlc.CreateMonitorResultParams{
		MonitorID:          monitorID,
		ConfirmingCheckIds: []int64{checkID},
		Result:             res.ResultPlaintext,
		Citations:          &res.Citations,
		Date:               &resultDate.Time,
		DatePastTenseVerb:  resultDatePastTenseVerb,
	}
}

func (s *Service) ActivateMonitorFromPreview(ctx context.Context, userID, monitorID int64) (*sqlc.Monitor, error) {
	monitor, err := s.GetMonitor(ctx, userID, monitorID)
	if err != nil {
		return nil, fmt.Errorf("getting monitor: %w", err)
	}

	if err = validateMonitorTransition(monitor.Status, sqlc.MonitorStatusActive); err != nil {
		return nil, err
	}

	return db.WithTxV(s.pool, ctx, func(tx pgx.Tx) (*sqlc.Monitor, error) {
		res, err := s.queries.GetLatestMonitorResult(ctx, tx, monitorID)
		if err != nil {
			return nil, fmt.Errorf("getting latest monitor result: %w", err)
		}

		if _, err := s.scheduleMonitorCheckTx(ctx, tx, monitor, res.CreatedAt.Add(monitorCheckFrequency)); err != nil {
			return nil, fmt.Errorf("scheduling check: %w", err)
		}

		monitor, err := s.updateMonitorStatus(ctx, tx, monitor, sqlc.MonitorStatusActive)
		if err != nil {
			return nil, fmt.Errorf("activating monitor: %w", err)
		}

		return monitor, nil
	})
}

type UpdateMonitorDraftParams struct {
	CommonParams
}

func NewUpdateMonitorDraftParams(mon *sqlc.Monitor) UpdateMonitorDraftParams {
	return UpdateMonitorDraftParams{
		CommonParams: CommonParams{
			Subject:      mon.Subject.String,
			Instructions: mon.Instructions.String,
		},
	}
}

func (s *Service) UpdateMonitorDraft(ctx context.Context, userID, monitorID int64, params UpdateMonitorDraftParams) (*sqlc.Monitor, error) {
	if err := s.validate.Struct(params); err != nil {
		return nil, err
	}

	mon, err := db.WithTxV(s.pool, ctx, func(tx pgx.Tx) (*sqlc.Monitor, error) {
		mon, err := s.queries.GetMonitor(ctx, tx, &sqlc.GetMonitorParams{
			UserID: userID,
			ID:     monitorID,
		})
		if err != nil {
			return nil, fmt.Errorf("getting monitor: %w", err)
		}

		if mon.Subject.String == params.Subject && mon.Instructions.String == params.Instructions {
			// don't bother updating if the monitor is similar
			return mon, nil
		}

		mon, err = s.updateMonitorStatus(ctx, tx, mon, sqlc.MonitorStatusValidating)
		if err != nil {
			return nil, err
		}

		mon, err = s.queries.UpdateMonitorDraft(ctx, tx, &sqlc.UpdateMonitorDraftParams{
			UserID:       mon.UserID,
			ID:           mon.ID,
			Subject:      pgtype.Text{String: params.Subject, Valid: true},
			Instructions: pgtype.Text{String: params.Instructions, Valid: true},
		})
		if err != nil {
			return nil, fmt.Errorf("updating monitor draft: %w", err)
		}

		if _, err = s.river.InsertTx(ctx, tx, ValidateMonitorArgs{
			UserID:    mon.UserID,
			MonitorID: mon.ID,
		}, nil); err != nil {
			return nil, fmt.Errorf("enqueuing validate monitor job: %w", err)
		}

		return mon, nil
	})

	if err != nil {
		return nil, err
	}

	return mon, nil
}

func (s *Service) deleteMonitorRelations(ctx context.Context, tx sqlc.DBTX, monitorID int64) error {
	if err := s.queries.DeleteMonitorChecks(ctx, tx, monitorID); err != nil {
		return fmt.Errorf("deleting monitor checks: %w", err)
	}
	if err := s.queries.DeleteMonitorResults(ctx, tx, monitorID); err != nil {
		return fmt.Errorf("deleting monitor results: %w", err)
	}
	if err := s.queries.DeleteMonitorCheckEventsForMonitor(ctx, tx, monitorID); err != nil {
		return fmt.Errorf("deleting monitor check events: %w", err)
	}
	return nil
}
