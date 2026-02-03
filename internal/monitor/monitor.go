package monitor

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alexpls/untils/internal/db"
	"github.com/alexpls/untils/internal/llm"
	"github.com/alexpls/untils/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func (s *Service) GetMonitor(ctx context.Context, userID, monitorID int64) (*models.Monitor, error) {
	monitor, err := s.queries.GetMonitor(ctx, s.db, &models.GetMonitorParams{
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

type MonitorCommonParams struct {
	Subject string `json:"subject" validate:"required,min=10,max=5000"`
}

type CreateMonitorParams struct {
	UserID int64 `validate:"required"`
	MonitorCommonParams
}

func (s *Service) CreateMonitor(ctx context.Context, params CreateMonitorParams) (*models.Monitor, error) {
	if err := s.validate.Struct(params); err != nil {
		return nil, err
	}

	return db.WithTxV(s.db, ctx, func(tx pgx.Tx) (*models.Monitor, error) {
		created, err := s.queries.CreateMonitor(ctx, tx, &models.CreateMonitorParams{
			UserID:  params.UserID,
			Subject: pgtype.Text{String: params.Subject, Valid: true},
		})
		if err != nil {
			return nil, fmt.Errorf("creating monitor: %w", err)
		}

		if err := s.enableAllNotifiers(ctx, tx, created); err != nil {
			return nil, fmt.Errorf("enabling notifiers: %w", err)
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
	err := s.queries.DeleteMonitor(ctx, s.db, &models.DeleteMonitorParams{
		UserID:    userID,
		MonitorID: monitorID,
	})
	if err != nil {
		return fmt.Errorf("deleting monitor: %w", err)
	}
	return nil
}

// TODO: the "validate" name should be more like triage now to align with package llm
func (s *Service) ValidateMonitor(ctx context.Context, monitor *models.Monitor) error {
	if monitor.Status == models.MonitorStatusActive {
		return fmt.Errorf("can't validate an active monitor")
	}

	monitor, err := s.updateMonitorStatus(ctx, s.db, monitor, models.MonitorStatusValidating)
	if err != nil {
		return fmt.Errorf("updating monitor status: %w", err)
	}

	// Get the latest result for feedback
	var userFeedback string
	latestResult, err := s.queries.GetLatestMonitorResult(ctx, s.db, monitor.ID)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("getting latest monitor result: %w", err)
		}
	} else if latestResult.Feedback.Valid {
		userFeedback = latestResult.Feedback.String
	}

	// Get previous results with checks for the triager
	prevs, err := s.queries.GetPreviousResultsWithCheck(ctx, s.db, monitor.ID)
	if err != nil {
		return fmt.Errorf("getting previous results: %w", err)
	}

	triage := s.llm.NewTriageWorkflow()

	trigageRes, err := triage.Run(ctx, &llm.CheckParams{
		Subject:         monitor.Subject.String,
		PreviousResults: prevs,
	})
	if err != nil {
		return fmt.Errorf("triage workflow: %w", err)
	}

	if !trigageRes.Approved {
		if err = s.queries.RejectMonitor(ctx, s.db, &models.RejectMonitorParams{
			ID:             monitor.ID,
			UserID:         monitor.UserID,
			RejectedReason: pgtype.Text{String: trigageRes.RejectedReason, Valid: true},
		}); err != nil {
			return fmt.Errorf("rejecting monitor: %w", err)
		}
		return nil
	}

	var check *models.MonitorCheck

	if err := db.WithTx(s.db, ctx, func(tx pgx.Tx) error {
		if err := s.validateMonitorsSameVersion(ctx, tx, monitor); err != nil {
			return err
		}

		// Delete old relations before creating a new check
		if err := s.deleteMonitorRelations(ctx, tx, monitor.ID); err != nil {
			return fmt.Errorf("deleting monitor relations: %w", err)
		}

		monitor, err = s.queries.UpdateMonitorStatus(ctx, tx, &models.UpdateMonitorStatusParams{
			ID:     monitor.ID,
			UserID: monitor.UserID,
			Status: models.MonitorStatusPreviewing,
		})
		if err != nil {
			return err
		}

		check, err = s.queries.CreateMonitorCheck(ctx, tx, &models.CreateMonitorCheckParams{
			MonitorID:    monitor.ID,
			Status:       models.MonitorCheckStatusScheduled,
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

	if err = s.PerformMonitorCheck(ctx, monitor.UserID, check, false, userFeedback); err != nil {
		return fmt.Errorf("performing monitor check: %w", err)
	}

	_, err = s.queries.UpdateMonitorToReady(ctx, s.db, &models.UpdateMonitorToReadyParams{
		MonitorID: monitor.ID,
		UserID:    monitor.UserID,
		Subject:   monitor.Subject,
	})

	return err
}

func CheckResultToCreateMonitorResultParams(monitorID, checkID int64, res *models.CheckResult) *models.CreateMonitorResultParams {
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

	return &models.CreateMonitorResultParams{
		MonitorID:          monitorID,
		ConfirmingCheckIds: []int64{checkID},
		Result:             res.ResultPlaintext,
		Citations:          &res.Citations,
		Date:               &resultDate.Time,
		DatePastTenseVerb:  resultDatePastTenseVerb,
	}
}

func (s *Service) ActivateMonitorFromPreview(ctx context.Context, user *models.User, monitorID int64) (*models.Monitor, error) {
	monitor, err := s.GetMonitor(ctx, user.ID, monitorID)
	if err != nil {
		return nil, fmt.Errorf("getting monitor: %w", err)
	}

	if err = validateMonitorTransition(monitor.Status, models.MonitorStatusActive); err != nil {
		return nil, err
	}

	return db.WithTxV(s.db, ctx, func(tx pgx.Tx) (*models.Monitor, error) {
		res, err := s.queries.GetLatestMonitorResult(ctx, tx, monitorID)
		if err != nil {
			return nil, fmt.Errorf("getting latest monitor result: %w", err)
		}

		fromTime := res.CreatedAt.In(user.Location())
		nextCheckTime := nextCheckTime(monitor.CheckFrequencyMinutes, fromTime)

		if _, err := s.scheduleMonitorCheckTx(ctx, tx, monitor, nextCheckTime); err != nil {
			return nil, fmt.Errorf("scheduling check: %w", err)
		}

		monitor, err := s.updateMonitorStatus(ctx, tx, monitor, models.MonitorStatusActive)
		if err != nil {
			return nil, fmt.Errorf("activating monitor: %w", err)
		}

		return monitor, nil
	})
}

type UpdateMonitorDraftParams struct {
	MonitorCommonParams
}

func NewUpdateMonitorDraftParams(mon *models.Monitor) UpdateMonitorDraftParams {
	return UpdateMonitorDraftParams{
		MonitorCommonParams: MonitorCommonParams{
			Subject: mon.Subject.String,
		},
	}
}

func (s *Service) updateMonitorDraftAndRevalidate(
	ctx context.Context,
	userID, monitorID int64,
	updater func(ctx context.Context, tx models.DBTX, mon *models.Monitor) (*models.Monitor, error),
) (*models.Monitor, error) {
	return db.WithTxV(s.db, ctx, func(tx pgx.Tx) (mon *models.Monitor, err error) {
		mon, err = s.queries.GetMonitor(ctx, tx, &models.GetMonitorParams{
			UserID: userID,
			ID:     monitorID,
		})
		if err != nil {
			return nil, fmt.Errorf("getting monitor: %w", err)
		}

		if mon, err = updater(ctx, tx, mon); err != nil {
			return nil, err
		}

		if mon, err = s.updateMonitorStatus(ctx, tx, mon, models.MonitorStatusValidating); err != nil {
			return nil, err
		}

		if _, err = s.river.InsertTx(ctx, tx, ValidateMonitorArgs{
			UserID:    mon.UserID,
			MonitorID: mon.ID,
		}, nil); err != nil {
			return nil, fmt.Errorf("enqueuing validate monitor job: %w", err)
		}

		return mon, err
	})
}

type UpdateMonitorFrequencyParams struct {
	CheckFrequencyMinutes int32 `json:"check_frequency_minutes"`
}

func (s *Service) UpdateMonitorFrequency(ctx context.Context, monitor *models.Monitor, params UpdateMonitorFrequencyParams) (*models.Monitor, error) {
	if err := validateFrequency(params.CheckFrequencyMinutes); err != nil {
		return nil, err
	}

	return db.WithTxV(s.db, ctx, func(tx pgx.Tx) (*models.Monitor, error) {
		// TODO: schedule next check whenever this is updated, too
		return s.queries.UpdateMonitorCheckFrequency(ctx, tx, &models.UpdateMonitorCheckFrequencyParams{
			MonitorID:             monitor.ID,
			CheckFrequencyMinutes: params.CheckFrequencyMinutes,
		})
	})
}

func (s *Service) UpdateMonitorDraft(ctx context.Context, userID, monitorID int64, params UpdateMonitorDraftParams) (*models.Monitor, error) {
	if err := s.validate.Struct(params); err != nil {
		return nil, err
	}

	return s.updateMonitorDraftAndRevalidate(ctx, userID, monitorID, func(ctx context.Context, tx models.DBTX, mon *models.Monitor) (*models.Monitor, error) {
		var err error

		if mon.Subject.String == params.Subject {
			// don't bother updating if the monitor is similar
			return mon, nil
		}

		if mon, err = s.queries.UpdateMonitorDraft(ctx, tx, &models.UpdateMonitorDraftParams{
			UserID:  mon.UserID,
			ID:      mon.ID,
			Subject: pgtype.Text{String: params.Subject, Valid: true},
		}); err != nil {
			return nil, fmt.Errorf("updating monitor draft: %w", err)
		}

		return mon, nil
	})
}

func (s *Service) deleteMonitorRelations(ctx context.Context, tx models.DBTX, monitorID int64) error {
	if err := s.queries.DeleteMonitorChecks(ctx, tx, monitorID); err != nil {
		return fmt.Errorf("deleting monitor checks: %w", err)
	}
	if err := s.queries.DeleteMonitorResults(ctx, tx, monitorID); err != nil {
		return fmt.Errorf("deleting monitor results: %w", err)
	}
	return nil
}

// SetMonitorPaused pauses or unpauses a monitor.
// When pausing, scheduled checks are deleted.
// When unpausing, a new check is scheduled for either now or when the next
// check would have been due if the monitor had never been paused, whichever is later.
func (s *Service) SetMonitorPaused(ctx context.Context, user *models.User, monitorID int64, paused bool) (*models.Monitor, error) {
	monitor, err := s.GetMonitor(ctx, user.ID, monitorID)
	if err != nil {
		return nil, fmt.Errorf("getting monitor: %w", err)
	}

	targetStatus := models.MonitorStatusActive
	if paused {
		targetStatus = models.MonitorStatusPaused
	}

	if err = validateMonitorTransition(monitor.Status, targetStatus); err != nil {
		return nil, err
	}

	return db.WithTxV(s.db, ctx, func(tx pgx.Tx) (*models.Monitor, error) {
		if paused {
			if err := s.queries.DeleteScheduledChecks(ctx, tx, monitorID); err != nil {
				return nil, fmt.Errorf("deleting scheduled checks: %w", err)
			}
		} else {
			nextCheckTime := nextCheckTime(monitor.CheckFrequencyMinutes, user.Now())
			if _, err := s.scheduleMonitorCheckTx(ctx, tx, monitor, nextCheckTime); err != nil {
				return nil, fmt.Errorf("scheduling check: %w", err)
			}
		}

		monitor, err = s.updateMonitorStatus(ctx, tx, monitor, targetStatus)
		if err != nil {
			return nil, fmt.Errorf("updating monitor status: %w", err)
		}

		return monitor, nil
	})
}
