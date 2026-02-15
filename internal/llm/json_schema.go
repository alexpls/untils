package llm

import (
	"maps"
	"reflect"
	"strings"
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
		required := make([]string, 0, numFields)

		for i := range numFields {
			field := t.Field(i)
			if !field.IsExported() {
				continue
			}

			fieldVal := v.Field(i)
			fieldName, omitEmpty, skip := jsonTagParts(field.Tag.Get("json"))
			if skip {
				continue
			}

			// Embedded fields without an explicit json name are flattened.
			if field.Anonymous && fieldName == "" {
				embeddedSchema := typeToSchema(field.Type, fieldVal)
				embeddedProps, _ := embeddedSchema["properties"].(map[string]any)
				embeddedRequired, _ := embeddedSchema["required"].([]string)

				maps.Copy(props, embeddedProps)
				required = append(required, embeddedRequired...)
				continue
			}

			if fieldName == "" {
				fieldName = field.Name
			}

			props[fieldName] = typeToSchema(field.Type, fieldVal)
			if !omitEmpty {
				required = append(required, fieldName)
			}
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
	case reflect.Pointer:
		elemType := t.Elem()
		elemValue := reflect.New(elemType).Elem()
		return typeToSchema(elemType, elemValue)
	}
	panic("unhandled type")
}

func jsonTagParts(tag string) (name string, omitEmpty bool, skip bool) {
	if tag == "-" {
		return "", false, true
	}

	if tag == "" {
		return "", false, false
	}

	parts := strings.Split(tag, ",")
	name = parts[0]
	for _, p := range parts[1:] {
		if p == "omitempty" {
			omitEmpty = true
		}
	}
	return name, omitEmpty, false
}
