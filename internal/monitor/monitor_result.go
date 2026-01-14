package monitor

import (
	"context"

	"github.com/alexpls/untils/internal/db/models"
	"github.com/jackc/pgx/v5/pgtype"
)

type CreateMonitorResultFeedbackParams struct {
	Feedback string `json:"feedback" validate:"required,min=3,max=2000"`
}

func (s *Service) CreateMonitorResultFeedback(ctx context.Context, result *models.MonitorResult, params CreateMonitorResultFeedbackParams) error {
	if err := s.validate.Struct(params); err != nil {
		return err
	}

	return s.queries.UpdateMonitorResultWithFeedback(ctx, s.pool, &models.UpdateMonitorResultWithFeedbackParams{
		MonitorResultID: result.ID,
		Feedback:        pgtype.Text{Valid: true, String: params.Feedback},
	})
}
