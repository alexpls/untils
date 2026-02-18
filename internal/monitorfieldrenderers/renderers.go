package monitorfieldrenderers

import (
	"strings"
	"time"

	"github.com/alexpls/untils/internal/datefmt"
	"github.com/alexpls/untils/internal/models"
)

type TextRenderer struct{}

func (TextRenderer) RenderField(ctx models.MonitorFieldsRenderContext, field models.MonitorUpdateField) string {
	return renderFieldValue(ctx, field)
}

func renderFieldValue(ctx models.MonitorFieldsRenderContext, field models.MonitorUpdateField) string {
	switch field.Type {
	case models.MonitorSchemaFieldTypeDate:
		return formatDateValue(ctx, field.Value)
	default:
		return field.Value
	}
}

func formatDateValue(ctx models.MonitorFieldsRenderContext, value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}

	// we don't use IsZero() on the parsed value because this is a quicker check
	// for zero values, and doesn't require us to additionally parse the date
	// in the UTC location to check that it's zero without any offset applied
	if value == "0001-01-01" {
		return ""
	}

	loc := models.LocationFromTimezone(ctx.Timezone)

	parsed, err := time.ParseInLocation("2006-01-02", trimmed, loc)
	if err != nil {
		return value
	}

	return parsed.In(loc).Format(datefmt.DateLayout)
}
