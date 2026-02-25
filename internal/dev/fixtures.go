package dev

import (
	"time"

	"github.com/alexpls/untils/internal/models"
	"github.com/alexpls/untils/internal/monitor"
	"github.com/jackc/pgx/v5/pgtype"
)

func fixtureInProgressCheckTimelineItemData() *monitor.InProgressCheckTimelineItemViewData {
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

func fixtureMonitorDraftPreviewingData() *monitor.MonitorDraftData {
	timelineItemData := fixtureInProgressCheckTimelineItemData()

	return &monitor.MonitorDraftData{
		Monitor: &models.Monitor{
			ID:      4242,
			Status:  models.MonitorStatusPreviewing,
			Subject: pgtype.Text{Valid: true, String: "Find major changes in the Kubernetes release notes"},
		},
		InProgressCheck:               timelineItemData.Check,
		InProgressCheckTimelineEvents: timelineItemData.TimelineEvents,
		Notifiers: []*monitor.MonitorNotifierViewData{
			{
				Integration: &models.UserIntegrationsRow{
					Name:       models.NotifierPushover,
					Configured: true,
				},
			},
			{
				Integration: &models.UserIntegrationsRow{
					Name:       models.NotifierEmail,
					Configured: true,
				},
			},
		},
	}
}

func fixtureMonitorDraftValidatingData() *monitor.MonitorDraftData {
	return &monitor.MonitorDraftData{
		Monitor: &models.Monitor{
			ID:      4244,
			Status:  models.MonitorStatusValidating,
			Subject: pgtype.Text{Valid: true, String: "Find major changes in the Kubernetes release notes"},
		},
		Notifiers: []*monitor.MonitorNotifierViewData{
			{
				Integration: &models.UserIntegrationsRow{
					Name:       models.NotifierPushover,
					Configured: true,
				},
			},
			{
				Integration: &models.UserIntegrationsRow{
					Name:       models.NotifierEmail,
					Configured: true,
				},
			},
		},
	}
}

func fixtureMonitorDraftRejectedData() *monitor.MonitorDraftData {
	return &monitor.MonitorDraftData{
		Monitor: &models.Monitor{
			ID:             4245,
			Status:         models.MonitorStatusRejected,
			Subject:        pgtype.Text{Valid: true, String: "Find major changes in the Kubernetes release notes"},
			RejectedReason: pgtype.Text{Valid: true, String: "This isn't an objective fact that can be monitored reliably. Lorem ipsum dolor sit amet. Lorem ipsum ipsum dolor sit amet."},
		},
		Notifiers: []*monitor.MonitorNotifierViewData{
			{
				Integration: &models.UserIntegrationsRow{
					Name:       models.NotifierPushover,
					Configured: true,
				},
			},
			{
				Integration: &models.UserIntegrationsRow{
					Name:       models.NotifierEmail,
					Configured: true,
				},
			},
		},
	}
}

func fixtureMonitorDraftReadyData() *monitor.MonitorDraftData {
	now := time.Now()

	return &monitor.MonitorDraftData{
		Monitor: &models.Monitor{
			ID:      4243,
			Status:  models.MonitorStatusReady,
			Subject: pgtype.Text{Valid: true, String: "Find major changes in the Kubernetes release notes"},
		},
		ResultPreview: &models.MonitorResult{
			ID:        1001,
			MonitorID: 4243,
			CreatedAt: now.Add(-2 * time.Minute),
			Headline:  "{{Summary}}",
			Subtitle:  "Source: {{Release notes URL}}",
			Data: models.MonitorUpdateData{
				Fields: models.MonitorUpdateFields{
					{
						MonitorSchemaField: models.MonitorSchemaField{
							Type: models.MonitorSchemaFieldTypeText,
							Name: "Summary",
						},
						Value: "Kubernetes v1.35 release notes published",
					},
					{
						MonitorSchemaField: models.MonitorSchemaField{
							Type: models.MonitorSchemaFieldTypeURL,
							Name: "Release notes URL",
						},
						Value: "https://kubernetes.io/releases/",
					},
				},
			},
		},
		Notifiers: []*monitor.MonitorNotifierViewData{
			{
				Integration: &models.UserIntegrationsRow{
					Name:       models.NotifierPushover,
					Configured: true,
				},
			},
			{
				Integration: &models.UserIntegrationsRow{
					Name:       models.NotifierEmail,
					Configured: true,
				},
			},
		},
	}
}
