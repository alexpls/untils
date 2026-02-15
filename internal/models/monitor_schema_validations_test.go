package models

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMonitorSchemaDataValidate(t *testing.T) {
	makeFields := func(n int) MonitorSchemaFields {
		fields := make(MonitorSchemaFields, 0, n)
		for i := range n {
			fields = append(fields, MonitorSchemaField{
				Type: MonitorSchemaFieldTypeText,
				Name: fieldNameForIndex(i),
			})
		}
		return fields
	}

	tests := []struct {
		name        string
		data        MonitorSchemaData
		errContains []string
	}{
		{
			name: "valid minimum schema",
			data: MonitorSchemaData{
				Headline: "{{Title}}",
				Fields: MonitorSchemaFields{
					{Type: MonitorSchemaFieldTypeText, Name: "Title"},
				},
			},
		},
		{
			name: "valid with optional subtitle",
			data: MonitorSchemaData{
				Headline: "{{Title}}",
				Subtitle: "Release: {{Release date}}",
				Fields: MonitorSchemaFields{
					{Type: MonitorSchemaFieldTypeText, Name: "Title"},
					{Type: MonitorSchemaFieldTypeDate, Name: "Release date"},
					{Type: MonitorSchemaFieldTypeURL, Name: "Link"},
				},
			},
		},
		{
			name: "invalid missing headline",
			data: MonitorSchemaData{
				Fields: MonitorSchemaFields{
					{Type: MonitorSchemaFieldTypeText, Name: "Title"},
				},
			},
			errContains: []string{"headline is required"},
		},
		{
			name: "invalid no fields",
			data: MonitorSchemaData{
				Headline: "{{Title}}",
			},
			errContains: []string{
				"at least one field is required",
				`headline references unknown field "Title"`,
			},
		},
		{
			name: "invalid too many fields",
			data: MonitorSchemaData{
				Headline: "{{Field01}}",
				Fields:   makeFields(11),
			},
			errContains: []string{"a maximum of 10 fields is allowed"},
		},
		{
			name: "invalid duplicate field names",
			data: MonitorSchemaData{
				Headline: "{{Title}}",
				Fields: MonitorSchemaFields{
					{Type: MonitorSchemaFieldTypeText, Name: "Title"},
					{Type: MonitorSchemaFieldTypeDate, Name: "Title"},
				},
			},
			errContains: []string{`duplicate field name "Title"`},
		},
		{
			name: "invalid more than one url field",
			data: MonitorSchemaData{
				Headline: "{{Title}}",
				Fields: MonitorSchemaFields{
					{Type: MonitorSchemaFieldTypeText, Name: "Title"},
					{Type: MonitorSchemaFieldTypeURL, Name: "Link one"},
					{Type: MonitorSchemaFieldTypeURL, Name: "Link two"},
				},
			},
			errContains: []string{"only one url field is allowed"},
		},
		{
			name: "invalid headline must reference field",
			data: MonitorSchemaData{
				Headline: "Static headline",
				Fields: MonitorSchemaFields{
					{Type: MonitorSchemaFieldTypeText, Name: "Title"},
				},
			},
			errContains: []string{"headline must reference at least one field"},
		},
		{
			name: "invalid headline unknown ref",
			data: MonitorSchemaData{
				Headline: "{{Unknown}}",
				Fields: MonitorSchemaFields{
					{Type: MonitorSchemaFieldTypeText, Name: "Title"},
				},
			},
			errContains: []string{`headline references unknown field "Unknown"`},
		},
		{
			name: "invalid subtitle static text",
			data: MonitorSchemaData{
				Headline: "{{Title}}",
				Subtitle: "Static subtitle",
				Fields: MonitorSchemaFields{
					{Type: MonitorSchemaFieldTypeText, Name: "Title"},
				},
			},
			errContains: []string{"subtitle must reference at least one field"},
		},
		{
			name: "invalid subtitle unknown ref",
			data: MonitorSchemaData{
				Headline: "{{Title}}",
				Subtitle: "Published {{Date}}",
				Fields: MonitorSchemaFields{
					{Type: MonitorSchemaFieldTypeText, Name: "Title"},
				},
			},
			errContains: []string{`subtitle references unknown field "Date"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.data.Validate()

			if len(tt.errContains) == 0 {
				require.NoError(t, err)
				return
			}

			require.Error(t, err)
			for _, part := range tt.errContains {
				require.ErrorContains(t, err, part)
			}
		})
	}
}

func TestMonitorSchemaFieldValidate(t *testing.T) {
	tests := []struct {
		name        string
		field       MonitorSchemaField
		errContains []string
	}{
		{
			name:  "valid text field",
			field: MonitorSchemaField{Type: MonitorSchemaFieldTypeText, Name: "Title"},
		},
		{
			name:        "invalid missing name",
			field:       MonitorSchemaField{Type: MonitorSchemaFieldTypeText, Name: ""},
			errContains: []string{"field name is required"},
		},
		{
			name:        "invalid type",
			field:       MonitorSchemaField{Type: "number", Name: "Count"},
			errContains: []string{`field type "number" is invalid`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.field.Validate()

			if len(tt.errContains) == 0 {
				require.NoError(t, err)
				return
			}

			require.Error(t, err)
			for _, part := range tt.errContains {
				require.ErrorContains(t, err, part)
			}
		})
	}
}

func TestMonitorUpdateDataValidate(t *testing.T) {
	makeFields := func(n int) []MonitorUpdateField {
		fields := make([]MonitorUpdateField, 0, n)
		for i := range n {
			fields = append(fields, MonitorUpdateField{
				MonitorSchemaField: MonitorSchemaField{
					Type: MonitorSchemaFieldTypeText,
					Name: fieldNameForIndex(i),
				},
				Value: "value",
			})
		}
		return fields
	}

	tests := []struct {
		name        string
		data        MonitorUpdateData
		errContains []string
	}{
		{
			name: "valid update",
			data: MonitorUpdateData{
				Fields: []MonitorUpdateField{
					{
						MonitorSchemaField: MonitorSchemaField{Type: MonitorSchemaFieldTypeText, Name: "Title"},
						Value:              "Mewgenics",
					},
					{
						MonitorSchemaField: MonitorSchemaField{Type: MonitorSchemaFieldTypeDate, Name: "Release date"},
						Value:              "2026-02-10",
					},
					{
						MonitorSchemaField: MonitorSchemaField{Type: MonitorSchemaFieldTypeURL, Name: "Link"},
						Value:              "https://www.ign.com/articles/mewgenics-review",
					},
				},
			},
		},
		{
			name: "invalid no fields",
			data: MonitorUpdateData{},
			errContains: []string{
				"at least one field is required",
			},
		},
		{
			name: "invalid too many fields",
			data: MonitorUpdateData{
				Fields: makeFields(11),
			},
			errContains: []string{"a maximum of 10 fields is allowed"},
		},
		{
			name: "invalid duplicate field names",
			data: MonitorUpdateData{
				Fields: []MonitorUpdateField{
					{
						MonitorSchemaField: MonitorSchemaField{Type: MonitorSchemaFieldTypeText, Name: "Title"},
						Value:              "one",
					},
					{
						MonitorSchemaField: MonitorSchemaField{Type: MonitorSchemaFieldTypeDate, Name: "Title"},
						Value:              "2026-01-01",
					},
				},
			},
			errContains: []string{`duplicate field name "Title"`},
		},
		{
			name: "invalid more than one url field",
			data: MonitorUpdateData{
				Fields: []MonitorUpdateField{
					{
						MonitorSchemaField: MonitorSchemaField{Type: MonitorSchemaFieldTypeURL, Name: "Link one"},
						Value:              "https://example.com/1",
					},
					{
						MonitorSchemaField: MonitorSchemaField{Type: MonitorSchemaFieldTypeURL, Name: "Link two"},
						Value:              "https://example.com/2",
					},
				},
			},
			errContains: []string{"only one url field is allowed"},
		},
		{
			name: "invalid nested field validation is surfaced",
			data: MonitorUpdateData{
				Fields: []MonitorUpdateField{
					{
						MonitorSchemaField: MonitorSchemaField{Type: MonitorSchemaFieldTypeDate, Name: "Release date"},
						Value:              "10/02/2026",
					},
				},
			},
			errContains: []string{`is not a valid date (YYYY-MM-DD)`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.data.Validate()

			if len(tt.errContains) == 0 {
				require.NoError(t, err)
				return
			}

			require.Error(t, err)
			for _, part := range tt.errContains {
				require.ErrorContains(t, err, part)
			}
		})
	}
}

func TestMonitorUpdateDataListValidateAgainstSchema(t *testing.T) {
	schema := MonitorSchemaData{
		Headline: "{{Title}}",
		Subtitle: "Release: {{Release date}}",
		Fields: MonitorSchemaFields{
			{Type: MonitorSchemaFieldTypeText, Name: "Title"},
			{Type: MonitorSchemaFieldTypeDate, Name: "Release date"},
			{Type: MonitorSchemaFieldTypeURL, Name: "Link"},
		},
	}

	tests := []struct {
		name        string
		updates     MonitorUpdateDataList
		errContains []string
	}{
		{
			name: "valid",
			updates: MonitorUpdateDataList{
				{
					Fields: []MonitorUpdateField{
						{
							MonitorSchemaField: MonitorSchemaField{Type: MonitorSchemaFieldTypeText, Name: "Title"},
							Value:              "Fear Inoculum",
						},
						{
							MonitorSchemaField: MonitorSchemaField{Type: MonitorSchemaFieldTypeDate, Name: "Release date"},
							Value:              "",
						},
						{
							MonitorSchemaField: MonitorSchemaField{Type: MonitorSchemaFieldTypeURL, Name: "Link"},
							Value:              "",
						},
					},
				},
			},
		},
		{
			name: "invalid missing field",
			updates: MonitorUpdateDataList{
				{
					Fields: []MonitorUpdateField{
						{
							MonitorSchemaField: MonitorSchemaField{Type: MonitorSchemaFieldTypeText, Name: "Title"},
							Value:              "Fear Inoculum",
						},
						{
							MonitorSchemaField: MonitorSchemaField{Type: MonitorSchemaFieldTypeDate, Name: "Release date"},
							Value:              "",
						},
					},
				},
			},
			errContains: []string{"expected 3 fields, got 2"},
		},
		{
			name: "invalid unknown field",
			updates: MonitorUpdateDataList{
				{
					Fields: []MonitorUpdateField{
						{
							MonitorSchemaField: MonitorSchemaField{Type: MonitorSchemaFieldTypeText, Name: "Title"},
							Value:              "Fear Inoculum",
						},
						{
							MonitorSchemaField: MonitorSchemaField{Type: MonitorSchemaFieldTypeDate, Name: "Release date"},
							Value:              "",
						},
						{
							MonitorSchemaField: MonitorSchemaField{Type: MonitorSchemaFieldTypeURL, Name: "URL"},
							Value:              "",
						},
					},
				},
			},
			errContains: []string{`unknown field "URL"`},
		},
		{
			name: "invalid wrong field type",
			updates: MonitorUpdateDataList{
				{
					Fields: []MonitorUpdateField{
						{
							MonitorSchemaField: MonitorSchemaField{Type: MonitorSchemaFieldTypeText, Name: "Title"},
							Value:              "Fear Inoculum",
						},
						{
							MonitorSchemaField: MonitorSchemaField{Type: MonitorSchemaFieldTypeText, Name: "Release date"},
							Value:              "",
						},
						{
							MonitorSchemaField: MonitorSchemaField{Type: MonitorSchemaFieldTypeURL, Name: "Link"},
							Value:              "",
						},
					},
				},
			},
			errContains: []string{`field "Release date" has type "text", expected "date"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.updates.ValidateAgainstSchema(schema)

			if len(tt.errContains) == 0 {
				require.NoError(t, err)
				return
			}

			require.Error(t, err)
			for _, part := range tt.errContains {
				require.ErrorContains(t, err, part)
			}
		})
	}
}

func TestMonitorUpdateFieldValidate(t *testing.T) {
	tests := []struct {
		name        string
		field       MonitorUpdateField
		errContains []string
	}{
		{
			name: "valid text value",
			field: MonitorUpdateField{
				MonitorSchemaField: MonitorSchemaField{Type: MonitorSchemaFieldTypeText, Name: "Title"},
				Value:              "Mewgenics",
			},
		},
		{
			name: "valid date value",
			field: MonitorUpdateField{
				MonitorSchemaField: MonitorSchemaField{Type: MonitorSchemaFieldTypeDate, Name: "Release date"},
				Value:              "2026-12-31",
			},
		},
		{
			name: "valid empty date value",
			field: MonitorUpdateField{
				MonitorSchemaField: MonitorSchemaField{Type: MonitorSchemaFieldTypeDate, Name: "Release date"},
				Value:              "",
			},
		},
		{
			name: "valid url value",
			field: MonitorUpdateField{
				MonitorSchemaField: MonitorSchemaField{Type: MonitorSchemaFieldTypeURL, Name: "Link"},
				Value:              "https://example.com/path",
			},
		},
		{
			name: "valid empty url value",
			field: MonitorUpdateField{
				MonitorSchemaField: MonitorSchemaField{Type: MonitorSchemaFieldTypeURL, Name: "Link"},
				Value:              "",
			},
		},
		{
			name: "invalid empty value",
			field: MonitorUpdateField{
				MonitorSchemaField: MonitorSchemaField{Type: MonitorSchemaFieldTypeText, Name: "Title"},
				Value:              "",
			},
			errContains: []string{"field value is required"},
		},
		{
			name: "invalid date format",
			field: MonitorUpdateField{
				MonitorSchemaField: MonitorSchemaField{Type: MonitorSchemaFieldTypeDate, Name: "Release date"},
				Value:              "2026-2-1",
			},
			errContains: []string{`is not a valid date (YYYY-MM-DD)`},
		},
		{
			name: "invalid url",
			field: MonitorUpdateField{
				MonitorSchemaField: MonitorSchemaField{Type: MonitorSchemaFieldTypeURL, Name: "Link"},
				Value:              "not-a-url",
			},
			errContains: []string{`is not a valid URL`},
		},
		{
			name: "invalid non-http url scheme",
			field: MonitorUpdateField{
				MonitorSchemaField: MonitorSchemaField{Type: MonitorSchemaFieldTypeURL, Name: "Link"},
				Value:              "ftp://example.com/resource",
			},
			errContains: []string{`must use http or https`},
		},
		{
			name: "invalid schema field validation is included",
			field: MonitorUpdateField{
				MonitorSchemaField: MonitorSchemaField{Type: "number", Name: ""},
				Value:              "123",
			},
			errContains: []string{
				"field name is required",
				`field type "number" is invalid`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.field.Validate()

			if len(tt.errContains) == 0 {
				require.NoError(t, err)
				return
			}

			require.Error(t, err)
			for _, part := range tt.errContains {
				require.ErrorContains(t, err, part)
			}
		})
	}
}

func fieldNameForIndex(i int) string {
	return "Field" + twoDigit(i+1)
}

func twoDigit(n int) string {
	if n < 10 {
		return "0" + string(rune('0'+n))
	}
	if n < 100 {
		return string(rune('0'+(n/10)%10)) + string(rune('0'+n%10))
	}
	return "99"
}
