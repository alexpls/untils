package webhook

import (
	"encoding/json"
	"testing"

	"github.com/alexpls/untils/internal/models"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func TestNewMessageMonitorNewResults(t *testing.T) {
	t.Parallel()

	payload, err := MarshalMessageMonitorNewResults(
		models.Monitor{ID: 42, Status: models.MonitorStatusActive, Subject: pgtype.Text{String: "Example monitor", Valid: true}},
		[]models.MonitorResult{
			{
				ID:        101,
				MonitorID: 42,
				Headline:  "{{Title}}",
				Subtitle:  "Released at {{Link}}",
				Data: models.MonitorUpdateData{Fields: models.MonitorUpdateFields{
					{
						MonitorSchemaField: models.MonitorSchemaField{Type: models.MonitorSchemaFieldTypeText, Name: "Title"},
						Value:              "New value",
					},
					{
						MonitorSchemaField: models.MonitorSchemaField{Type: models.MonitorSchemaFieldTypeURL, Name: "Link"},
						Value:              "https://example.com/new",
					},
				}},
			},
			{ID: 102, MonitorID: 42, Headline: "Another value"},
		},
		models.MonitorResult{ID: 100, MonitorID: 42, Headline: "Old value"},
	)
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(payload, &got))
	require.Equal(t, "webhook_message", got["type"])

	message := got["message"].(map[string]any)
	require.Equal(t, "new_results", message["type"])
	require.Equal(t, "active", message["monitor"].(map[string]any)["status"])
	require.Equal(t, "Example monitor", message["monitor"].(map[string]any)["subject"])
	newResults := message["new_results"].([]any)
	require.Len(t, newResults, 2)
	require.Equal(t, float64(101), newResults[0].(map[string]any)["id"])
	require.Equal(t, "New value", newResults[0].(map[string]any)["headline"])
	require.Equal(t, "Released at https://example.com/new", newResults[0].(map[string]any)["subtitle"])
	require.Equal(t, float64(102), newResults[1].(map[string]any)["id"])
	require.Equal(t, float64(100), message["old_result"].(map[string]any)["id"])
}
