package monitor

import (
	"context"

	"github.com/alexpls/untils/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type CreateMonitorResultFeedbackParams struct {
	Feedback string `json:"feedback" validate:"required,min=3,max=2000"`
}

func (s *Service) CreateMonitorResultFeedback(ctx context.Context, userID int64, result *models.MonitorResult, params CreateMonitorResultFeedbackParams) error {
	if err := s.validate.Struct(params); err != nil {
		return err
	}

	if _, err := s.updateMonitorDraftAndRevalidate(ctx, userID, result.MonitorID, func(ctx context.Context, tx pgx.Tx, mon *models.Monitor) (*models.Monitor, error) {
		if err := s.queries.UpdateMonitorResultWithFeedback(ctx, tx, &models.UpdateMonitorResultWithFeedbackParams{
			MonitorResultID: result.ID,
			Feedback:        pgtype.Text{Valid: true, String: params.Feedback},
		}); err != nil {
			return nil, err
		}
		return mon, nil
	}); err != nil {
		return err
	}

	return nil
}
