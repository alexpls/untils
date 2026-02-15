package llm

import (
	"testing"

	"github.com/alexpls/untils/internal/models"
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
