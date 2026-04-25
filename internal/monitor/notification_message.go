package monitor

import (
	"context"
	"fmt"

	"github.com/alexpls/untils/internal/models"
	"github.com/alexpls/untils/internal/notifications"
)

func newResultsNotificationMessage(monitor models.Monitor, newValues []models.MonitorResult, oldValue models.MonitorResult) notifications.MonitorNewResults {
	return notifications.MonitorNewResults{
		Monitor:    monitor,
		NewResults: newValues,
		OldResult:  oldValue,
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
