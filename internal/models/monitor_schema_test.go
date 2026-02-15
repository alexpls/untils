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
