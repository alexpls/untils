package notifications

import (
	"context"
	"testing"

	"github.com/alexpls/untils/internal/models"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func TestRenderMonitorNewResult(t *testing.T) {
	t.Parallel()

	renderConfig := RenderConfig{BaseURL: "https://untils.example.com"}

	rendered, err := RenderMonitorNewResult(context.Background(), renderConfig, MonitorNewResult{
		Monitor: models.Monitor{Subject: pgtype.Text{String: "Example monitor", Valid: true}},
		New: models.MonitorResult{
			MonitorID: 42,
			Headline:  "{{Title}}",
			Subtitle:  "Released at {{Link}}",
			Data: models.MonitorUpdateData{
				Fields: models.MonitorUpdateFields{
					{
						MonitorSchemaField: models.MonitorSchemaField{
							Type: models.MonitorSchemaFieldTypeText,
							Name: "Title",
						},
						Value: "new value",
					},
					{
						MonitorSchemaField: models.MonitorSchemaField{
							Type: models.MonitorSchemaFieldTypeURL,
							Name: "Link",
						},
						Value: "https://example.com/new",
					},
				},
			},
		},
		Old: models.MonitorResult{
			Headline: "old value",
			Subtitle: "Old subtitle",
			Data: models.MonitorUpdateData{
				Fields: models.MonitorUpdateFields{
					{
						MonitorSchemaField: models.MonitorSchemaField{
							Type: models.MonitorSchemaFieldTypeURL,
							Name: "Archive",
						},
						Value: "https://example.com/old",
					},
				},
			},
		},
	})
	require.NoError(t, err)

	require.Equal(t, "New result: Example monitor", rendered.Email.Subject)
	require.Equal(t, "New result: Example monitor", rendered.Pushover.Title)
	require.Equal(t, "New: new value\n\nOld: old value", rendered.Email.TextBody)
	require.Equal(t, "New: new value\n\nOld: old value", rendered.Pushover.Message)
	require.Contains(t, rendered.Email.HTMLBody, "Example monitor")
	require.Contains(t, rendered.Email.HTMLBody, "new value")
	require.Contains(t, rendered.Email.HTMLBody, "old value")
	require.Contains(t, rendered.Email.HTMLBody, "Released at https://example.com/new")
	require.Contains(t, rendered.Email.HTMLBody, "Old subtitle")
	require.Contains(t, rendered.Email.HTMLBody, "https://example.com/new")
	require.Contains(t, rendered.Email.HTMLBody, "https://example.com/old")
	require.Contains(t, rendered.Email.HTMLBody, "https://untils.example.com/")
	require.Contains(t, rendered.Email.HTMLBody, "https://untils.example.com/assets/images/logo")
	require.Contains(t, rendered.Email.HTMLBody, "alt=\"untils\"")
	require.Contains(t, rendered.Email.HTMLBody, "Correct or hide it")
	require.Contains(t, rendered.Email.HTMLBody, "https://untils.example.com/app/monitors/42")
}

func TestEmailTemplateStore(t *testing.T) {
	t.Parallel()

	store := NewEmailTemplateStore(RenderConfig{BaseURL: "https://untils.example.com"})

	templates := store.Templates()
	require.Len(t, templates, 1)
	require.Equal(t, "new_result", templates[0].Key)

	tmpl, ok := store.Template("new_result")
	require.True(t, ok)
	require.Equal(t, "New result", tmpl.Name)

	_, ok = store.Template("missing")
	require.False(t, ok)
}
