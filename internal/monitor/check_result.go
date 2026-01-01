package monitor

import (
	"context"
	"fmt"

	"github.com/alexpls/untils/internal/db/sqlc"
)

func (s *Service) ListMonitorResults(ctx context.Context, monitor *sqlc.Monitor) ([]*sqlc.MonitorResult, error) {
	results, err := s.queries.ListMonitorResults(ctx, s.pool, monitor.ID)
	if err != nil {
		return nil, fmt.Errorf("listing monitor check results: %w", err)
	}

	return results, nil
}
