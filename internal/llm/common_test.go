package llm

import (
	"testing"
	"time"

	"github.com/alexpls/untils/internal/models"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func TestCheckParamsUserMessageStringIncludesSchema(t *testing.T) {
	msg := (CheckParams{
		Subject: "Latest album by Tool",
		Schema: models.MonitorSchemaData{
			Headline: "{{Album name}}",
			Subtitle: "Release date: {{Release date}}",
			Fields: models.MonitorSchemaFields{
				{Type: models.MonitorSchemaFieldTypeText, Name: "Album name"},
				{Type: models.MonitorSchemaFieldTypeDate, Name: "Release date"},
				{Type: models.MonitorSchemaFieldTypeURL, Name: "Link"},
			},
		},
	}).UserMessageString()

	require.Contains(t, msg, "## Subject:\nLatest album by Tool")
	require.Contains(t, msg, "## Monitor schema:")
	require.Contains(t, msg, `"headline":"{{Album name}}"`)
	require.Contains(t, msg, `"name":"Album name"`)
}

func TestCheckParamsUserMessageStringWithoutSchema(t *testing.T) {
	msg := (CheckParams{
		Subject: "Latest album by Tool",
	}).UserMessageString()

	require.Contains(t, msg, "## Subject:\nLatest album by Tool")
	require.NotContains(t, msg, "## Monitor schema:")
}

func TestCheckParamsPreviousResultsStringIncludesJSONPayload(t *testing.T) {
	citations := models.Citations{
		{URL: "https://example.com/item"},
	}

	msg := (CheckParams{
		Schema: models.MonitorSchemaData{
			Headline: "{{Title}}",
			Fields: models.MonitorSchemaFields{
				{Type: models.MonitorSchemaFieldTypeText, Name: "Title"},
			},
		},
		PreviousResults: []*models.GetPreviousResultsWithCheckRow{
			{
				MonitorResult: models.MonitorResult{
					Data: models.MonitorUpdateData{
						Fields: models.MonitorUpdateFields{
							{
								MonitorSchemaField: models.MonitorSchemaField{
									Type: models.MonitorSchemaFieldTypeText,
									Name: "Title",
								},
								Value: "Fear Inoculum",
							},
						},
					},
					LastConfirmedAt: time.Date(2026, 2, 16, 15, 4, 0, 0, time.UTC),
					Feedback:        pgtype.Text{String: "Use canonical source pages", Valid: true},
					Citations:       &citations,
				},
			},
		},
	}).PreviousResultsString()

	require.Contains(t, msg, `"schema":`)
	require.Contains(t, msg, `"data":{"fields":[`)
	require.Contains(t, msg, `"latest_check_ran_at":`)
	require.Contains(t, msg, `"user_feedback":"Use canonical source pages"`)
	require.Contains(t, msg, `"sources_used":["https://example.com/item"]`)
}
