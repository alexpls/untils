package models

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

func (f MonitorSchemaFields) GetValue(name string) string {
	for _, field := range f {
		if field.Name == name {
			return string(field.Type)
		}
	}
	return ""
}

func (f MonitorUpdateFields) GetValue(name string) string {
	for _, field := range f {
		if field.Name == name {
			return field.Value
		}
	}
	return ""
}
