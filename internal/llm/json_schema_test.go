package llm

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type Nested struct {
	NestedString string `json:"nested_string"`
}

type Embedded struct {
	EmbeddedString string `json:"embedded_string"`
}

type TestStruct struct {
	BoolField   bool     `json:"bool_field"`
	StringField string   `json:"string_field"`
	StructField Nested   `json:"struct_field"`
	SliceField  []Nested `json:"slice_field"`
}

type TestStructWithEmbedded struct {
	Embedded
	Name string `json:"name"`
}

func TestJsonSchema(t *testing.T) {
	s := TestStruct{}

	schema := jsonSchema(s)

	require.Equal(t, map[string]any{
		"type":                 "object",
		"required":             []string{"bool_field", "string_field", "struct_field", "slice_field"},
		"additionalProperties": false,
		"properties": map[string]any{
			"bool_field": map[string]any{
				"type": "boolean",
			},
			"string_field": map[string]any{
				"type": "string",
			},
			"struct_field": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"nested_string": map[string]any{
						"type": "string",
					},
				},
				"required":             []string{"nested_string"},
				"additionalProperties": false,
			},
			"slice_field": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"nested_string": map[string]any{
							"type": "string",
						},
					},
					"required":             []string{"nested_string"},
					"additionalProperties": false,
				},
			},
		},
	}, schema)
}

func TestJsonSchemaWithEmbeddedStruct(t *testing.T) {
	s := TestStructWithEmbedded{}

	schema := jsonSchema(s)

	require.Equal(t, map[string]any{
		"type":                 "object",
		"required":             []string{"embedded_string", "name"},
		"additionalProperties": false,
		"properties": map[string]any{
			"embedded_string": map[string]any{
				"type": "string",
			},
			"name": map[string]any{
				"type": "string",
			},
		},
	}, schema)
}
