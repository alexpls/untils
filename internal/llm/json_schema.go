package llm

import (
	"reflect"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
)

func jsonSchema(s any) map[string]any {
	structType := reflect.TypeOf(s)
	if structType.Kind() != reflect.Struct {
		panic("didn't get a struct")
	}
	structValue := reflect.ValueOf(s)
	return typeToSchema(structType, structValue)
}

func jsonSchemaResponse(s any) responses.ResponseTextConfigParam {
	name := reflect.TypeOf(s).Name()
	return responses.ResponseTextConfigParam{
		Format: responses.ResponseFormatTextConfigUnionParam{
			OfJSONSchema: &responses.ResponseFormatTextJSONSchemaConfigParam{
				Name:   name,
				Strict: openai.Bool(true),
				Schema: jsonSchema(s),
			},
		},
	}
}

func typeToSchema(t reflect.Type, v reflect.Value) map[string]any {
	switch t.Kind() {
	case reflect.Bool:
		return map[string]any{
			"type": "boolean",
		}
	case reflect.String:
		return map[string]any{
			"type": "string",
		}
	case reflect.Struct:
		numFields := t.NumField()
		props := make(map[string]any)
		required := make([]string, numFields)

		for i := range numFields {
			field := t.Field(i)
			fieldVal := v.Field(i)
			fieldTag := field.Tag.Get("json")
			props[fieldTag] = typeToSchema(field.Type, fieldVal)
			required[i] = fieldTag
		}

		return map[string]any{
			"type":                 "object",
			"additionalProperties": false,
			"properties":           props,
			"required":             required,
		}
	case reflect.Slice:
		elemType := t.Elem()
		elemValue := reflect.New(elemType).Elem()

		return map[string]any{
			"type":  "array",
			"items": typeToSchema(elemType, elemValue),
		}
	}
	panic("unhandled type")
}
