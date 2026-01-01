package monitor

import (
	"context"
	"encoding/json"

	"github.com/alexpls/untils_go/internal/db/sqlc"
)

type CreateMonitorCheckEventParams struct {
	Kind    sqlc.MonitorCheckEventKind
	Details sqlc.MonitorCheckEventDetails
}

func (s *Service) CreateMonitorCheckEvent(ctx context.Context, monitorCheckID int64, params CreateMonitorCheckEventParams) (*sqlc.MonitorCheckEvent, error) {
	j, err := json.Marshal(params.Details)
	if err != nil {
		return nil, err
	}

	return s.queries.CreateMonitorCheckEvent(ctx, s.pool, &sqlc.CreateMonitorCheckEventParams{
		MonitorCheckID: monitorCheckID,
		Kind:           params.Kind,
		Details:        j,
	})
}
