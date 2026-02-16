package monitorfieldrenderers

import (
	"testing"

	"github.com/alexpls/untils/internal/models"
	"github.com/stretchr/testify/require"
)

func TestTextRendererRenderFieldDate(t *testing.T) {
	field := models.MonitorUpdateField{
		MonitorSchemaField: models.MonitorSchemaField{
			Type: models.MonitorSchemaFieldTypeDate,
			Name: "Release date",
		},
		Value: "2019-08-30",
	}

	got := TextRenderer{}.RenderField(models.MonitorFieldsRenderContext{
		Timezone: "America/New_York",
	}, field)
	require.Equal(t, "Aug 30, 2019", got)
}

func TestTextRendererRenderFieldDateInvalidValue(t *testing.T) {
	field := models.MonitorUpdateField{
		MonitorSchemaField: models.MonitorSchemaField{
			Type: models.MonitorSchemaFieldTypeDate,
			Name: "Release date",
		},
		Value: "not-a-date",
	}

	got := TextRenderer{}.RenderField(models.MonitorFieldsRenderContext{
		Timezone: "America/New_York",
	}, field)
	require.Equal(t, "not-a-date", got)
}
