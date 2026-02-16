package models

import (
	"strings"
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

func (d MonitorSchemaData) RenderHeadline(
	fields MonitorUpdateFields,
	renderer MonitorFieldsRenderer,
	renderCtx MonitorFieldsRenderContext,
) (string, error) {
	return fields.RenderTemplate(d.Headline, renderer, renderCtx)
}

func (d MonitorSchemaData) RenderSubtitle(
	fields MonitorUpdateFields,
	renderer MonitorFieldsRenderer,
	renderCtx MonitorFieldsRenderContext,
) (string, error) {
	if strings.TrimSpace(d.Subtitle) == "" {
		return "", nil
	}
	return fields.RenderTemplate(d.Subtitle, renderer, renderCtx)
}

func (f MonitorSchemaFields) GetValue(name string) string {
	for _, field := range f {
		if field.Name == name {
			return string(field.Type)
		}
	}
	return ""
}
