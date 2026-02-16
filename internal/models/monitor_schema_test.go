package models

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMonitorSchemaFieldsGetValue(t *testing.T) {
	fields := MonitorSchemaFields{
		{Type: MonitorSchemaFieldTypeText, Name: "Title"},
		{Type: MonitorSchemaFieldTypeDate, Name: "Release date"},
		{Type: MonitorSchemaFieldTypeURL, Name: "Link"},
	}

	require.Equal(t, string(MonitorSchemaFieldTypeText), fields.GetValue("Title"))
	require.Equal(t, string(MonitorSchemaFieldTypeDate), fields.GetValue("Release date"))
	require.Equal(t, string(MonitorSchemaFieldTypeURL), fields.GetValue("Link"))
	require.Equal(t, "", fields.GetValue("Unknown"))
}

func TestMonitorUpdateFieldsGetValue(t *testing.T) {
	fields := MonitorUpdateFields{
		{
			MonitorSchemaField: MonitorSchemaField{Type: MonitorSchemaFieldTypeText, Name: "Title"},
			Value:              "Fear Inoculum",
		},
		{
			MonitorSchemaField: MonitorSchemaField{Type: MonitorSchemaFieldTypeDate, Name: "Release date"},
			Value:              "2019-08-30",
		},
		{
			MonitorSchemaField: MonitorSchemaField{Type: MonitorSchemaFieldTypeURL, Name: "Link"},
			Value:              "https://en.wikipedia.org/wiki/Fear_Inoculum",
		},
	}

	require.Equal(t, "Fear Inoculum", fields.GetValue("Title"))
	require.Equal(t, "2019-08-30", fields.GetValue("Release date"))
	require.Equal(t, "https://en.wikipedia.org/wiki/Fear_Inoculum", fields.GetValue("Link"))
	require.Equal(t, "", fields.GetValue("Unknown"))
}

func TestMonitorUpdateFieldsResolveTemplate(t *testing.T) {
	fields := MonitorUpdateFields{
		{
			MonitorSchemaField: MonitorSchemaField{Type: MonitorSchemaFieldTypeText, Name: "Title"},
			Value:              "Fear Inoculum",
		},
		{
			MonitorSchemaField: MonitorSchemaField{Type: MonitorSchemaFieldTypeDate, Name: "Release date"},
			Value:              "2019-08-30",
		},
	}

	resolved, err := fields.RenderTemplate(
		"{{Title}} released on {{Release date}}",
		testMonitorFieldsRenderer{render: func(field MonitorUpdateField) string { return field.Value }},
		MonitorFieldsRenderContext{},
	)
	require.NoError(t, err)
	require.Equal(t, "Fear Inoculum released on 2019-08-30", resolved)
}

func TestMonitorUpdateFieldsResolveTemplateMissingField(t *testing.T) {
	fields := MonitorUpdateFields{
		{
			MonitorSchemaField: MonitorSchemaField{Type: MonitorSchemaFieldTypeText, Name: "Title"},
			Value:              "Fear Inoculum",
		},
	}

	_, err := fields.RenderTemplate(
		"{{Release date}}",
		testMonitorFieldsRenderer{render: func(field MonitorUpdateField) string { return field.Value }},
		MonitorFieldsRenderContext{},
	)
	require.ErrorContains(t, err, `missing value for field "Release date"`)
}

func TestMonitorSchemaDataRenderHeadlineAndSubtitle(t *testing.T) {
	schema := MonitorSchemaData{
		Headline: "{{Title}}",
		Subtitle: "Released: {{Release date}}",
	}
	fields := MonitorUpdateFields{
		{
			MonitorSchemaField: MonitorSchemaField{Type: MonitorSchemaFieldTypeText, Name: "Title"},
			Value:              "Fear Inoculum",
		},
		{
			MonitorSchemaField: MonitorSchemaField{Type: MonitorSchemaFieldTypeDate, Name: "Release date"},
			Value:              "2019-08-30",
		},
	}

	headline, err := schema.RenderHeadline(fields, testMonitorFieldsRenderer{render: func(field MonitorUpdateField) string { return field.Value }}, MonitorFieldsRenderContext{})
	require.NoError(t, err)
	require.Equal(t, "Fear Inoculum", headline)

	subtitle, err := schema.RenderSubtitle(fields, testMonitorFieldsRenderer{render: func(field MonitorUpdateField) string { return field.Value }}, MonitorFieldsRenderContext{})
	require.NoError(t, err)
	require.Equal(t, "Released: 2019-08-30", subtitle)
}

func TestMonitorSchemaDataRenderSubtitleEmpty(t *testing.T) {
	schema := MonitorSchemaData{
		Headline: "{{Title}}",
		Subtitle: "",
	}

	subtitle, err := schema.RenderSubtitle(MonitorUpdateFields{}, testMonitorFieldsRenderer{render: func(field MonitorUpdateField) string { return field.Value }}, MonitorFieldsRenderContext{})
	require.NoError(t, err)
	require.Equal(t, "", subtitle)
}

func TestMonitorSchemaDataRenderWithCustomRenderer(t *testing.T) {
	schema := MonitorSchemaData{
		Headline: "{{Title}}",
		Subtitle: "Released: {{Release date}}",
	}
	fields := MonitorUpdateFields{
		{
			MonitorSchemaField: MonitorSchemaField{Type: MonitorSchemaFieldTypeText, Name: "Title"},
			Value:              "Fear Inoculum",
		},
		{
			MonitorSchemaField: MonitorSchemaField{Type: MonitorSchemaFieldTypeDate, Name: "Release date"},
			Value:              "2019-08-30",
		},
	}

	renderer := testMonitorFieldsRenderer{
		render: func(field MonitorUpdateField) string {
			if field.Type == MonitorSchemaFieldTypeDate {
				return "Aug 30, 2019"
			}
			return field.Value
		},
	}

	headline, err := schema.RenderHeadline(fields, renderer, MonitorFieldsRenderContext{})
	require.NoError(t, err)
	require.Equal(t, "Fear Inoculum", headline)

	subtitle, err := schema.RenderSubtitle(fields, renderer, MonitorFieldsRenderContext{})
	require.NoError(t, err)
	require.Equal(t, "Released: Aug 30, 2019", subtitle)
}

func TestMonitorUpdateFieldsEqual(t *testing.T) {
	base := MonitorUpdateFields{
		{
			MonitorSchemaField: MonitorSchemaField{Type: MonitorSchemaFieldTypeText, Name: "Title"},
			Value:              "Fear Inoculum",
		},
		{
			MonitorSchemaField: MonitorSchemaField{Type: MonitorSchemaFieldTypeDate, Name: "Release date"},
			Value:              "2019-08-30",
		},
		{
			MonitorSchemaField: MonitorSchemaField{Type: MonitorSchemaFieldTypeURL, Name: "Link"},
			Value:              "https://en.wikipedia.org/wiki/Fear_Inoculum",
		},
	}

	reordered := MonitorUpdateFields{
		base[2],
		base[0],
		base[1],
	}

	differentValue := MonitorUpdateFields{
		base[0],
		base[1],
		{
			MonitorSchemaField: MonitorSchemaField{Type: MonitorSchemaFieldTypeURL, Name: "Link"},
			Value:              "https://example.com",
		},
	}

	missingField := MonitorUpdateFields{
		base[0],
		base[1],
	}

	// Missing Title field with an empty value should not be treated as equal.
	missingFieldWithEmptyValue := MonitorUpdateFields{
		{
			MonitorSchemaField: MonitorSchemaField{Type: MonitorSchemaFieldTypeDate, Name: "Release date"},
			Value:              "2019-08-30",
		},
		{
			MonitorSchemaField: MonitorSchemaField{Type: MonitorSchemaFieldTypeURL, Name: "Link"},
			Value:              "https://en.wikipedia.org/wiki/Fear_Inoculum",
		},
		{
			MonitorSchemaField: MonitorSchemaField{Type: MonitorSchemaFieldTypeText, Name: "Different"},
			Value:              "",
		},
	}

	require.True(t, MonitorUpdateFieldsEqual(base, base))
	require.True(t, MonitorUpdateFieldsEqual(base, reordered))
	require.False(t, MonitorUpdateFieldsEqual(base, differentValue))
	require.False(t, MonitorUpdateFieldsEqual(base, missingField))
	require.False(t, MonitorUpdateFieldsEqual(base, missingFieldWithEmptyValue))
}

type testMonitorFieldsRenderer struct {
	render func(field MonitorUpdateField) string
}

func (r testMonitorFieldsRenderer) RenderField(_ MonitorFieldsRenderContext, field MonitorUpdateField) string {
	return r.render(field)
}
