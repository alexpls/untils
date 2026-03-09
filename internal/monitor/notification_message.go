package monitor

import (
	"context"
	"fmt"

	"github.com/alexpls/untils/internal/models"
	"github.com/alexpls/untils/internal/notifications"
)

func newResultNotificationMessage(monitor models.Monitor, newValue, oldValue models.MonitorResult) notifications.MonitorNewResult {
	return notifications.MonitorNewResult{
		Monitor: monitor,
		New:     newValue,
		Old:     oldValue,
	}
}

func (s *Service) previousVisibleNotificationResult(
	ctx context.Context,
	monitorID int64,
	currentResultID int64,
) (models.MonitorResult, error) {
	results, err := s.queries.ListMonitorResults(ctx, s.db, monitorID)
	if err != nil {
		return models.MonitorResult{}, fmt.Errorf("listing monitor results: %w", err)
	}

	for i, candidate := range results {
		if candidate.ID != currentResultID {
			continue
		}
		if i+1 >= len(results) {
			return emptyNotificationResult(), nil
		}

		return *results[i+1], nil
	}

	return emptyNotificationResult(), nil
}

func emptyNotificationResult() models.MonitorResult {
	return models.MonitorResult{Headline: "(none)"}
}
