package dev

import (
	"time"

	"github.com/alexpls/untils/internal/models"
	"github.com/alexpls/untils/internal/monitor"
	"github.com/jackc/pgx/v5/pgtype"
)

func stubbedInProgressCheckTimelineItemData() *monitor.InProgressCheckTimelineItemViewData {
	now := time.Now()

	return &monitor.InProgressCheckTimelineItemViewData{
		Check: &models.MonitorCheck{},
		TimelineEvents: []*models.GetTimelineEventsBySourceIDRow{
			{
				Name:      "search_request",
				Arguments: `{"query":"list of the best example websites"}`,
				At:        pgtype.Timestamptz{Time: now.Add(-60 * time.Second), Valid: true},
			},
			{
				Name:      "search_request",
				Arguments: `{"query":"text text text text text text text text text text"}`,
				At:        pgtype.Timestamptz{Time: now.Add(-50 * time.Second), Valid: true},
			},
			{
				Name:      "browser_navigate",
				Arguments: `{"url":"https://example.com/pricing"}`,
				At:        pgtype.Timestamptz{Time: now.Add(-40 * time.Second), Valid: true},
			},
			{
				Name:      "browser_click",
				Arguments: "",
				At:        pgtype.Timestamptz{Time: now.Add(-30 * time.Second), Valid: true},
			},
		},
	}
}
