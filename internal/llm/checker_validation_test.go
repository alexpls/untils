package llm

import (
	"testing"

	"github.com/alexpls/untils/internal/models"
	"github.com/stretchr/testify/require"
)

func TestFirstCheckUpdateCountMismatch(t *testing.T) {
	tests := []struct {
		name         string
		res          *models.CheckResultWithSchema
		isFirstCheck bool
		want         bool
	}{
		{
			name:         "first check success one update",
			res:          checkResult(true, sampleUpdate("Title", "A")),
			isFirstCheck: true,
			want:         false,
		},
		{
			name:         "first check success no updates",
			res:          checkResult(true),
			isFirstCheck: true,
			want:         true,
		},
		{
			name:         "first check success multiple updates",
			res:          checkResult(true, sampleUpdate("Title", "A"), sampleUpdate("Title", "B")),
			isFirstCheck: true,
			want:         true,
		},
		{
			name:         "first check failure allows no updates",
			res:          checkResult(false),
			isFirstCheck: true,
			want:         false,
		},
		{
			name:         "non first check multiple updates allowed",
			res:          checkResult(true, sampleUpdate("Title", "A"), sampleUpdate("Title", "B")),
			isFirstCheck: false,
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := firstCheckUpdateCountMismatch(tt.res, tt.isFirstCheck)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestDuplicateUpdatesMismatch(t *testing.T) {
	tests := []struct {
		name string
		res  *models.CheckResultWithSchema
		want bool
	}{
		{
			name: "single update",
			res:  checkResult(true, sampleUpdate("Title", "A")),
			want: false,
		},
		{
			name: "duplicate updates",
			res: checkResult(
				true,
				models.MonitorUpdateData{
					Fields: []models.MonitorUpdateField{
						{
							MonitorSchemaField: models.MonitorSchemaField{Type: models.MonitorSchemaFieldTypeText, Name: "Title"},
							Value:              "A",
						},
						{
							MonitorSchemaField: models.MonitorSchemaField{Type: models.MonitorSchemaFieldTypeURL, Name: "Link"},
							Value:              "https://example.com/a",
						},
					},
				},
				models.MonitorUpdateData{
					Fields: []models.MonitorUpdateField{
						{
							MonitorSchemaField: models.MonitorSchemaField{Type: models.MonitorSchemaFieldTypeURL, Name: "Link"},
							Value:              "https://example.com/a",
						},
						{
							MonitorSchemaField: models.MonitorSchemaField{Type: models.MonitorSchemaFieldTypeText, Name: "Title"},
							Value:              "A",
						},
					},
				},
			),
			want: true,
		},
		{
			name: "distinct updates",
			res:  checkResult(true, sampleUpdate("Title", "A"), sampleUpdate("Title", "B")),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := duplicateUpdatesMismatch(tt.res)
			require.Equal(t, tt.want, got)
		})
	}
}

func checkResult(success bool, updates ...models.MonitorUpdateData) *models.CheckResultWithSchema {
	return &models.CheckResultWithSchema{
		CheckResultBase: models.CheckResultBase{
			Success: success,
			Updates: models.MonitorUpdateDataList(updates),
		},
	}
}

func sampleUpdate(fieldName, value string) models.MonitorUpdateData {
	return models.MonitorUpdateData{
		Fields: []models.MonitorUpdateField{
			{
				MonitorSchemaField: models.MonitorSchemaField{
					Type: models.MonitorSchemaFieldTypeText,
					Name: fieldName,
				},
				Value: value,
			},
		},
	}
}
