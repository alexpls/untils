package models

type MonitorSchemaData struct {
	Fields MonitorSchemaFields `json:"fields"`
}

func (d MonitorSchemaData) Zero() bool {
	return len(d.Fields) == 0
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

func (f MonitorSchemaFields) GetValue(name string) string {
	for _, field := range f {
		if field.Name == name {
			return string(field.Type)
		}
	}
	return ""
}
