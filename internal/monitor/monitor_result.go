package monitor

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alexpls/untils/internal/db"
	"github.com/alexpls/untils/internal/models"
	"github.com/alexpls/untils/internal/pagination"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type CreateMonitorResultCorrectionParams struct {
	Correction string `json:"correction" validate:"required"`
}

func (s *Service) ListLatestResults(ctx context.Context, userID int64, limit int32) ([]*models.ListLatestVisibleResultsForUserRow, error) {
	return s.queries.ListLatestVisibleResultsForUser(ctx, s.db, &models.ListLatestVisibleResultsForUserParams{
		UserID:      userID,
		ResultLimit: limit,
	})
}

type ListResultsParams struct {
	UserID     int64
	MonitorID  int64
	Pagination pagination.Pagination
}

type ListResultsPage struct {
	Results    []*models.MonitorResult
	Pagination pagination.Pagination
}

func (s *Service) ListResults(ctx context.Context, params ListResultsParams) (*ListResultsPage, error) {
	if _, err := s.GetMonitor(ctx, params.UserID, params.MonitorID); err != nil {
		return nil, err
	}

	pag := params.Pagination
	results, err := s.queries.ListMonitorResultsPage(ctx, s.db, &models.ListMonitorResultsPageParams{
		MonitorID: params.MonitorID,
		UserID:    params.UserID,
		RowOffset: pag.Offset64(),
		PageSize:  int64(pag.PageSizeWithPeek()),
	})
	if err != nil {
		return nil, fmt.Errorf("listing monitor results: %w", err)
	}

	results, pag = pagination.Peek(results, pag)

	return &ListResultsPage{Results: results, Pagination: pag}, nil
}

func (s *Service) CreateMonitorResultCorrection(ctx context.Context, userID int64, result *models.MonitorResult, params CreateMonitorResultCorrectionParams) error {
	if err := s.validate.Struct(params); err != nil {
		return err
	}

	mon, err := s.GetMonitor(ctx, userID, result.MonitorID)
	if err != nil {
		return err
	}

	switch mon.Status {
	case models.MonitorStatusReady:
		_, err := s.updateMonitorDraftAndRevalidate(ctx, userID, mon.ID, func(ctx context.Context, tx models.DBTX, mon *models.Monitor) (*models.Monitor, error) {
			if err := s.assertLatestVisibleResult(ctx, tx, result); err != nil {
				return nil, err
			}

			if err := s.applyMonitorResultCorrection(ctx, tx, result.ID, params.Correction); err != nil {
				return nil, err
			}

			return mon, nil
		})
		return err
	case models.MonitorStatusActive:
		return db.WithTx(s.db, ctx, func(tx pgx.Tx) error {
			if err := s.assertLatestVisibleResult(ctx, tx, result); err != nil {
				return err
			}

			if err := s.applyMonitorResultCorrection(ctx, tx, result.ID, params.Correction); err != nil {
				return err
			}

			if err := s.cancelMonitorJobsTx(ctx, tx, mon.ID); err != nil {
				return fmt.Errorf("cancelling monitor jobs: %w", err)
			}

			if err := s.queries.DeleteStaleChecks(ctx, tx, mon.ID); err != nil {
				return fmt.Errorf("deleting stale checks: %w", err)
			}

			if _, err := s.scheduleMonitorCheckTx(ctx, tx, mon, time.Now()); err != nil {
				return fmt.Errorf("scheduling replacement check: %w", err)
			}

			return nil
		})
	default:
		return db.WithTx(s.db, ctx, func(tx pgx.Tx) error {
			if err := s.assertLatestVisibleResult(ctx, tx, result); err != nil {
				return err
			}

			return s.applyMonitorResultCorrection(ctx, tx, result.ID, params.Correction)
		})
	}
}

func (s *Service) AssertMonitorResultCorrectionAllowed(ctx context.Context, result *models.MonitorResult) error {
	return s.assertLatestVisibleResult(ctx, s.db, result)
}

func (s *Service) HideMonitorResult(ctx context.Context, userID int64, result *models.MonitorResult) error {
	mon, err := s.GetMonitor(ctx, userID, result.MonitorID)
	if err != nil {
		return err
	}

	switch mon.Status {
	case models.MonitorStatusActive, models.MonitorStatusPaused:
	default:
		return ErrMonitorResultHideNotAllowed
	}

	return s.queries.HideMonitorResult(ctx, s.db, result.ID)
}

func (s *Service) applyMonitorResultCorrection(ctx context.Context, tx models.DBTX, resultID int64, correction string) error {
	if err := s.queries.UpdateMonitorResultCorrection(ctx, tx, &models.UpdateMonitorResultCorrectionParams{
		MonitorResultID:  resultID,
		ResultCorrection: pgtype.Text{Valid: correction != "", String: correction},
	}); err != nil {
		return fmt.Errorf("updating monitor result correction: %w", err)
	}

	if err := s.queries.HideMonitorResult(ctx, tx, resultID); err != nil {
		return fmt.Errorf("hiding monitor result: %w", err)
	}

	return nil
}

func (s *Service) assertLatestVisibleResult(ctx context.Context, tx models.DBTX, result *models.MonitorResult) error {
	latestResult, err := s.queries.GetLatestMonitorResult(ctx, tx, result.MonitorID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrMonitorResultCorrectionNotAllowed
		}

		return fmt.Errorf("getting latest visible monitor result: %w", err)
	}

	if !result.CanApplyCorrection(latestResult) {
		return ErrMonitorResultCorrectionNotAllowed
	}

	return nil
}
