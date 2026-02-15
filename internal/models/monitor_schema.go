package models

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type MonitorSchemaData struct {
	Headline string               `json:"headline"`
	Subtitle string               `json:"subtitle"`
	Fields   []MonitorSchemaField `json:"fields"`
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

type MonitorUpdateData struct {
	Fields []MonitorUpdateField `json:"fields"`
}

type MonitorUpdateField struct {
	MonitorSchemaField
	Value string `json:"value"`
}

const maxMonitorSchemaFields = 10

// TODO: actually implement template parsing somewhere else, and then use
// it here for our validation, too.
var templateFieldRefRegexp = regexp.MustCompile(`{{\s*([^{}]+?)\s*}}`)

func (d MonitorSchemaData) Validate() error {
	var errs []error

	fieldNameSet, fieldErrs := validateMonitorSchemaFields(d.Fields)
	errs = append(errs, fieldErrs...)

	if strings.TrimSpace(d.Headline) == "" {
		errs = append(errs, errors.New("headline is required"))
	} else {
		errs = append(errs, validateTemplateRefs("headline", d.Headline, fieldNameSet)...)
	}

	if strings.TrimSpace(d.Subtitle) != "" {
		errs = append(errs, validateTemplateRefs("subtitle", d.Subtitle, fieldNameSet)...)
	}

	return errors.Join(errs...)
}

func (d MonitorSchemaField) Validate() error {
	var errs []error

	if strings.TrimSpace(d.Name) == "" {
		errs = append(errs, errors.New("field name is required"))
	}

	switch d.Type {
	case MonitorSchemaFieldTypeText, MonitorSchemaFieldTypeDate, MonitorSchemaFieldTypeURL:
	default:
		errs = append(errs, fmt.Errorf("field type %q is invalid", d.Type))
	}

	return errors.Join(errs...)
}

func (d MonitorUpdateData) Validate() error {
	var errs []error

	_, fieldErrs := validateMonitorUpdateFields(d.Fields)
	errs = append(errs, fieldErrs...)

	return errors.Join(errs...)
}

func (d MonitorUpdateField) Validate() error {
	var errs []error

	if err := d.MonitorSchemaField.Validate(); err != nil {
		errs = append(errs, err)
	}

	value := strings.TrimSpace(d.Value)
	if value == "" {
		errs = append(errs, errors.New("field value is required"))
	}

	switch d.Type {
	case MonitorSchemaFieldTypeDate:
		if value != "" {
			parsed, err := time.Parse("2006-01-02", value)
			if err != nil || parsed.Format("2006-01-02") != value {
				errs = append(errs, fmt.Errorf("field value %q is not a valid date (YYYY-MM-DD)", d.Value))
			}
		}
	case MonitorSchemaFieldTypeURL:
		if value != "" {
			parsed, err := url.Parse(value)
			if err != nil || parsed.Scheme == "" || parsed.Host == "" {
				errs = append(errs, fmt.Errorf("field value %q is not a valid URL", d.Value))
			} else if parsed.Scheme != "http" && parsed.Scheme != "https" {
				errs = append(errs, fmt.Errorf("field value %q must use http or https", d.Value))
			}
		}
	}

	return errors.Join(errs...)
}

func validateMonitorSchemaFields(fields []MonitorSchemaField) (map[string]struct{}, []error) {
	return validateMonitorFields(
		fields,
		func(field MonitorSchemaField) MonitorSchemaField { return field },
		func(field MonitorSchemaField) error { return field.Validate() },
	)
}

func validateMonitorUpdateFields(fields []MonitorUpdateField) (map[string]struct{}, []error) {
	return validateMonitorFields(
		fields,
		func(field MonitorUpdateField) MonitorSchemaField { return field.MonitorSchemaField },
		func(field MonitorUpdateField) error { return field.Validate() },
	)
}

func validateMonitorFields[T any](
	fields []T,
	getSchemaField func(T) MonitorSchemaField,
	validateField func(T) error,
) (map[string]struct{}, []error) {
	var errs []error

	if len(fields) == 0 {
		errs = append(errs, errors.New("at least one field is required"))
	}

	if len(fields) > maxMonitorSchemaFields {
		errs = append(errs, fmt.Errorf("a maximum of %d fields is allowed", maxMonitorSchemaFields))
	}

	fieldNames := make(map[string]struct{}, len(fields))
	urlFieldCount := 0

	for i, field := range fields {
		if err := validateField(field); err != nil {
			errs = append(errs, fmt.Errorf("fields[%d]: %w", i, err))
		}

		schemaField := getSchemaField(field)
		name := strings.TrimSpace(schemaField.Name)
		if name != "" {
			if _, exists := fieldNames[name]; exists {
				errs = append(errs, fmt.Errorf("fields[%d]: duplicate field name %q", i, schemaField.Name))
			} else {
				fieldNames[name] = struct{}{}
			}
		}

		if schemaField.Type == MonitorSchemaFieldTypeURL {
			urlFieldCount++
		}
	}

	if urlFieldCount > 1 {
		errs = append(errs, errors.New("only one url field is allowed"))
	}

	return fieldNames, errs
}

func validateTemplateRefs(fieldLabel, template string, validFieldNames map[string]struct{}) []error {
	refs := templateFieldRefRegexp.FindAllStringSubmatch(template, -1)
	if len(refs) == 0 {
		return []error{fmt.Errorf("%s must reference at least one field", fieldLabel)}
	}

	var errs []error
	for _, ref := range refs {
		name := strings.TrimSpace(ref[1])
		if _, ok := validFieldNames[name]; !ok {
			errs = append(errs, fmt.Errorf("%s references unknown field %q", fieldLabel, name))
		}
	}

	return errs
}
