package models

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/alexpls/untils/internal/tinytemplate"
)

const maxMonitorSchemaFields = 10

func (d MonitorUpdateDataList) Validate() error {
	var errs []error

	for i, d := range d {
		if err := d.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("updates[%d]: %w", i, err))
		}
	}

	return errors.Join(errs...)
}

func (d MonitorUpdateDataList) ValidateAgainstSchema(schema MonitorSchemaData) error {
	if schema.Zero() {
		return nil
	}

	var errs []error

	schemaByName := make(map[string]MonitorSchemaFieldType, len(schema.Fields))
	for _, schemaField := range schema.Fields {
		schemaByName[strings.TrimSpace(schemaField.Name)] = schemaField.Type
	}

	for i, update := range d {
		if len(update.Fields) != len(schema.Fields) {
			errs = append(errs, fmt.Errorf("updates[%d]: expected %d fields, got %d", i, len(schema.Fields), len(update.Fields)))
			continue
		}

		seen := make(map[string]struct{}, len(update.Fields))
		for j, field := range update.Fields {
			name := strings.TrimSpace(field.Name)
			expectedType, ok := schemaByName[name]
			if !ok {
				errs = append(errs, fmt.Errorf("updates[%d].fields[%d]: unknown field %q", i, j, field.Name))
				continue
			}
			if field.Type != expectedType {
				errs = append(errs, fmt.Errorf("updates[%d].fields[%d]: field %q has type %q, expected %q", i, j, field.Name, field.Type, expectedType))
			}
			seen[name] = struct{}{}
		}

		for _, schemaField := range schema.Fields {
			name := strings.TrimSpace(schemaField.Name)
			if _, ok := seen[name]; !ok {
				errs = append(errs, fmt.Errorf("updates[%d]: missing field %q", i, schemaField.Name))
			}
		}
	}

	return errors.Join(errs...)
}

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
	// TODO: validate against schema to make sure it matches well
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
	if d.Type == MonitorSchemaFieldTypeText && value == "" {
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

func validateMonitorSchemaFields(fields MonitorSchemaFields) (map[string]struct{}, []error) {
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
	tt, err := tinytemplate.Parse(template)
	if err != nil {
		return []error{fmt.Errorf("%s template is invalid: %w", fieldLabel, err)}
	}

	refs := tt.References()
	if len(refs) == 0 {
		return []error{fmt.Errorf("%s must reference at least one field", fieldLabel)}
	}

	var errs []error
	for _, name := range refs {
		if _, ok := validFieldNames[name]; !ok {
			errs = append(errs, fmt.Errorf("%s references unknown field %q", fieldLabel, name))
		}
	}

	return errs
}
