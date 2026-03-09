package monitor

import (
	"context"
	"fmt"

	"github.com/alexpls/untils/internal/models"
	"github.com/alexpls/untils/internal/monitorfieldrenderers"
	"github.com/alexpls/untils/internal/notifications"
)

func newResultNotificationMessage(subject, newValue, oldValue string) notifications.MonitorNewResult {
	return notifications.MonitorNewResult{
		Subject: subject,
		New:     newValue,
		Old:     oldValue,
	}
}

func renderNotificationHeadline(result *models.MonitorResult, timezone string) (string, error) {
	renderer := monitorfieldrenderers.TextRenderer{}
	renderCtx := models.MonitorFieldsRenderContext{Timezone: timezone}

	headline, err := result.RenderHeadline(renderer, renderCtx)
	if err != nil {
		return "", fmt.Errorf("rendering notification headline: %w", err)
	}

	return headline, nil
}

func (s *Service) previousVisibleNotificationHeadline(
	ctx context.Context,
	monitorID int64,
	currentResultID int64,
	timezone string,
) (string, error) {
	results, err := s.queries.ListMonitorResults(ctx, s.db, monitorID)
	if err != nil {
		return "", fmt.Errorf("listing monitor results: %w", err)
	}

	for i, candidate := range results {
		if candidate.ID != currentResultID {
			continue
		}
		if i+1 >= len(results) {
			return "(none)", nil
		}

		return renderNotificationHeadline(results[i+1], timezone)
	}

	return "(none)", nil
}
