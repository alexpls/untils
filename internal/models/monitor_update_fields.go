package models

import (
	"fmt"
	"strings"

	"github.com/alexpls/untils/internal/tinytemplate"
)

type MonitorUpdateData struct {
	Headline string              `json:"headline"`
	Subtitle string              `json:"subtitle"`
	Fields   MonitorUpdateFields `json:"fields"`
}

type MonitorUpdateDataList []MonitorUpdateData

type MonitorUpdateField struct {
	MonitorSchemaField
	Value string `json:"value"`
}

type MonitorUpdateFields []MonitorUpdateField

func (f MonitorUpdateFields) GetValue(name string) string {
	value, ok := f.LookupValue(name)
	if ok {
		return value
	}
	return ""
}

func (f MonitorUpdateFields) LookupValue(name string) (string, bool) {
	field, ok := f.LookupField(name)
	if !ok {
		return "", false
	}
	return field.Value, true
}

func (f MonitorUpdateFields) LookupField(name string) (MonitorUpdateField, bool) {
	for _, field := range f {
		if field.Name == name {
			return field, true
		}
	}
	return MonitorUpdateField{}, false
}

type MonitorFieldsRenderContext struct {
	Timezone string
}

type MonitorFieldsRenderer interface {
	RenderField(ctx MonitorFieldsRenderContext, field MonitorUpdateField) string
}

func (f MonitorUpdateFields) RenderTemplate(
	template string,
	renderer MonitorFieldsRenderer,
	renderCtx MonitorFieldsRenderContext,
) (string, error) {
	if renderer == nil {
		return "", fmt.Errorf("monitor fields renderer is required")
	}

	tt, err := tinytemplate.Parse(template)
	if err != nil {
		return "", err
	}

	return tt.RenderFunc(func(name string) (string, bool) {
		field, ok := f.LookupField(name)
		if !ok {
			return "", false
		}
		return renderer.RenderField(renderCtx, field), true
	})
}

func (f MonitorUpdateFields) MustRenderTemplate(
	template string,
	renderer MonitorFieldsRenderer,
	renderCtx MonitorFieldsRenderContext,
) string {
	rendered, err := f.RenderTemplate(template, renderer, renderCtx)
	if err != nil {
		panic(err)
	}
	return rendered
}

func (d MonitorUpdateData) RenderHeadline(
	renderer MonitorFieldsRenderer,
	renderCtx MonitorFieldsRenderContext,
) (string, error) {
	return d.Fields.RenderTemplate(d.Headline, renderer, renderCtx)
}

func (d MonitorUpdateData) RenderSubtitle(
	renderer MonitorFieldsRenderer,
	renderCtx MonitorFieldsRenderContext,
) (string, error) {
	if strings.TrimSpace(d.Subtitle) == "" {
		return "", nil
	}
	return d.Fields.RenderTemplate(d.Subtitle, renderer, renderCtx)
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
