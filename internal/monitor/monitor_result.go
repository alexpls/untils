package monitor

import (
	"context"

	"github.com/alexpls/untils/internal/models"
	"github.com/jackc/pgx/v5/pgtype"
)

type CreateMonitorResultFeedbackParams struct {
	Feedback string `json:"feedback" validate:"required,min=3,max=2000"`
}

func (s *Service) CreateMonitorResultFeedback(ctx context.Context, userID int64, result *models.MonitorResultsWithLatestCheck, params CreateMonitorResultFeedbackParams) error {
	if err := s.validate.Struct(params); err != nil {
		return err
	}

	mon, err := s.GetMonitor(ctx, userID, result.MonitorID)
	if err != nil {
		return err
	}

	updater := func(ctx context.Context, tx models.DBTX, mon *models.Monitor) (*models.Monitor, error) {
		err := s.queries.UpdateMonitorResultWithFeedback(ctx, tx, &models.UpdateMonitorResultWithFeedbackParams{
			MonitorResultID: result.ID,
			Feedback:        pgtype.Text{Valid: true, String: params.Feedback},
		})
		return mon, err
	}

	switch mon.Status {
	case models.MonitorStatusReady:
		if _, err := s.updateMonitorDraftAndRevalidate(ctx, userID, mon.ID, updater); err != nil {
			return err
		}
	default:
		if _, err := updater(ctx, s.db, mon); err != nil {
			return err
		}
	}

	return nil
}
