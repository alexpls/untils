package monitor

import (
	"context"
	"encoding/json"

	"github.com/alexpls/untils/internal/models"
)

type CreateMonitorCheckEventParams struct {
	Kind    models.MonitorCheckEventKind
	Details models.MonitorCheckEventDetails
}

func (s *Service) CreateMonitorCheckEvent(ctx context.Context, check *models.MonitorCheck, params CreateMonitorCheckEventParams) (*models.MonitorCheckEvent, error) {
	j, err := json.Marshal(params.Details)
	if err != nil {
		return nil, err
	}

	return s.queries.CreateMonitorCheckEvent(ctx, s.pool, &models.CreateMonitorCheckEventParams{
		MonitorID:      check.MonitorID,
		MonitorCheckID: check.ID,
		Kind:           params.Kind,
		Details:        j,
	})
}
