package llm

import (
	"reflect"
)

func jsonSchema(s any) map[string]any {
	structType := reflect.TypeOf(s)
	if structType.Kind() != reflect.Struct {
		panic("didn't get a struct")
	}
	structValue := reflect.ValueOf(s)
	return typeToSchema(structType, structValue)
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
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Float32, reflect.Float64:
		return map[string]any{
			"type": "number",
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
