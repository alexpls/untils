package models

import (
	"strings"

	"github.com/alexpls/untils/internal/tinytemplate"
)

type MonitorSchemaData struct {
	Headline string              `json:"headline"`
	Subtitle string              `json:"subtitle"`
	Fields   MonitorSchemaFields `json:"fields"`
}

func (d MonitorSchemaData) Zero() bool {
	return d.Headline == "" && d.Subtitle == "" && len(d.Fields) == 0
}

type MonitorSchemaFieldType string

const (
	MonitorSchemaFieldTypeText MonitorSchemaFieldType = "text"
	MonitorSchemaFieldTypeDate MonitorSchemaFieldType = "date"
	MonitorSchemaFieldTypeURL  MonitorSchemaFieldType = "url"
)

type MonitorSchemaField struct {
	Type MonitorSchemaFieldType `json:"type"`
	Name string                 `json:"name"`
}

type MonitorSchemaFields []MonitorSchemaField

type MonitorUpdateData struct {
	Fields MonitorUpdateFields `json:"fields"`
}

type MonitorUpdateDataList []MonitorUpdateData

type MonitorUpdateField struct {
	MonitorSchemaField
	Value string `json:"value"`
}

type MonitorUpdateFields []MonitorUpdateField

func (d MonitorSchemaData) RenderHeadline(fields MonitorUpdateFields) (string, error) {
	return fields.ResolveTemplate(d.Headline)
}

func (d MonitorSchemaData) RenderSubtitle(fields MonitorUpdateFields) (string, error) {
	if strings.TrimSpace(d.Subtitle) == "" {
		return "", nil
	}
	return fields.ResolveTemplate(d.Subtitle)
}

func (f MonitorSchemaFields) GetValue(name string) string {
	for _, field := range f {
		if field.Name == name {
			return string(field.Type)
		}
	}
	return ""
}

func (f MonitorUpdateFields) GetValue(name string) string {
	value, ok := f.LookupValue(name)
	if ok {
		return value
	}
	return ""
}

func (f MonitorUpdateFields) LookupValue(name string) (string, bool) {
	for _, field := range f {
		if field.Name == name {
			return field.Value, true
		}
	}
	return "", false
}

func (f MonitorUpdateFields) ResolveTemplate(template string) (string, error) {
	tt, err := tinytemplate.Parse(template)
	if err != nil {
		return "", err
	}

	return tt.RenderFunc(func(name string) (string, bool) {
		return f.LookupValue(name)
	})
}

func MonitorUpdateFieldsEqual(a, b MonitorUpdateFields) bool {
	if len(a) != len(b) {
		return false
	}

	valuesByField := make(map[string]string, len(a))
	for _, field := range a {
		valuesByField[monitorUpdateFieldKey(field)] = field.Value
	}

	for _, field := range b {
		value, ok := valuesByField[monitorUpdateFieldKey(field)]
		if !ok || value != field.Value {
			return false
		}
	}

	return true
}

func monitorUpdateFieldKey(field MonitorUpdateField) string {
	return string(field.Type) + "\x00" + field.Name
}
