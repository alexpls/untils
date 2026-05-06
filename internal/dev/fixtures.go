package dev

import (
	"strings"
	"time"

	"github.com/alexpls/untils/internal/models"
	"github.com/alexpls/untils/internal/monitor"
	"github.com/jackc/pgx/v5/pgtype"
)

type monitorListCardFixture struct {
	Title   string
	Monitor *models.ListMonitorsWithResultsRow
}

func fixtureMonitorListCardData() []monitorListCardFixture {
	now := time.Now()

	return []monitorListCardFixture{
		{
			Title:   "Active with compact text",
			Monitor: fixtureMonitorListCard(1001, models.MonitorStatusActive, "Kubernetes release notes", "Kubernetes v1.35 release notes published", "Source: release notes", now.Add(-14*time.Minute), now.Add(46*time.Minute), false),
		},
		{
			Title: "Paused with long subject",
			Monitor: fixtureMonitorListCard(
				1002,
				models.MonitorStatusPaused,
				"Monitor the public pricing page for a deeply nested enterprise plan where several limits, add-ons, and regional tax notes can change independently",
				"The enterprise plan remains unchanged",
				"Last checked from https://example.com/pricing/enterprise/regions/australia-and-new-zealand",
				now.Add(-7*time.Hour),
				now.Add(18*time.Hour),
				false,
			),
		},
		{
			Title: "Long headline and subtitle",
			Monitor: fixtureMonitorListCard(
				1003,
				models.MonitorStatusActive,
				"Track a government procurement notice",
				"The notice added a clarifying amendment about eligibility, submission attachments, late lodgement handling, and the required statutory declaration wording for applicants",
				"Amendment details were found in the downloadable addendum linked from the tender portal, with a long URL and a verbose document title",
				now.Add(-31*time.Minute),
				now.Add(29*time.Minute),
				false,
			),
		},
		{
			Title:   "Missing latest result",
			Monitor: fixtureMonitorListCard(1004, models.MonitorStatusActive, "First result pending", "", "", time.Time{}, now.Add(12*time.Minute), false),
		},
		{
			Title:   "Currently checking",
			Monitor: fixtureMonitorListCard(1005, models.MonitorStatusActive, "Track uptime incident page", "No open incidents", "Status page summary unchanged", now.Add(-2*time.Hour), time.Time{}, true),
		},
		{
			Title: "Unexpected single-word values",
			Monitor: fixtureMonitorListCard(
				1006,
				models.MonitorStatusActive,
				strings.Repeat("subject", 12),
				strings.Repeat("headline", 14),
				strings.Repeat("subtitle", 16),
				now.Add(-90*time.Second),
				now.Add(90*time.Second),
				false,
			),
		},
	}
}

func fixtureMonitorListCard(
	id int64,
	status models.MonitorStatus,
	subject string,
	headline string,
	subtitle string,
	latestResultCreatedAt time.Time,
	nextCheckScheduledFor time.Time,
	currentlyChecking bool,
) *models.ListMonitorsWithResultsRow {
	return &models.ListMonitorsWithResultsRow{
		MonitorID: id,
		Status:    status,
		Subject:   subject,
		Headline:  "{{Headline}}",
		Subtitle:  "{{Subtitle}}",
		Data: models.MonitorUpdateData{
			Fields: models.MonitorUpdateFields{
				{
					MonitorSchemaField: models.MonitorSchemaField{
						Type: models.MonitorSchemaFieldTypeText,
						Name: "Headline",
					},
					Value: headline,
				},
				{
					MonitorSchemaField: models.MonitorSchemaField{
						Type: models.MonitorSchemaFieldTypeText,
						Name: "Subtitle",
					},
					Value: subtitle,
				},
			},
		},
		LatestResultCreatedAt: latestResultCreatedAt,
		NextCheckScheduledFor: nextCheckScheduledFor,
		CurrentlyChecking:     currentlyChecking,
	}
}

func fixtureInProgressCheckTimelineItemData() *monitor.InProgressCheckTimelineItemViewData {
	now := time.Now()

	return &monitor.InProgressCheckTimelineItemViewData{
		Check: &models.MonitorCheck{},
		TimelineEvents: []*models.GetTimelineEventsBySourceIDRow{
			{
				Name:      models.LLMToolNameSearchRequest,
				Arguments: `{"query":"list of the best example websites"}`,
				At:        pgtype.Timestamptz{Time: now.Add(-60 * time.Second), Valid: true},
			},
			{
				Name:      models.LLMToolNameSearchRequest,
				Arguments: `{"query":"text text text text text text text text text text"}`,
				At:        pgtype.Timestamptz{Time: now.Add(-50 * time.Second), Valid: true},
			},
			{
				Name:      models.LLMToolNameBrowserNavigate,
				Arguments: `{"url":"https://example.com/pricing"}`,
				At:        pgtype.Timestamptz{Time: now.Add(-40 * time.Second), Valid: true},
			},
			{
				Name:      models.LLMToolNameBrowserClick,
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
