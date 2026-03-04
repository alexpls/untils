package llm

import (
	"encoding/json"
	"strings"

	"github.com/alexpls/untils/internal/models"
	"github.com/alexpls/untils/internal/must"
)

type CheckParams struct {
	UserID          int64
	MonitorID       int64
	MonitorCheckID  int64
	Timezone        string
	Subject         string
	PreviousResults []*models.GetPreviousResultsWithCheckRow
	Schema          models.MonitorSchemaData
}

func (c CheckParams) UserMessageString() string {
	tz := c.Timezone
	if tz == "" {
		tz = "UTC"
	}

	var b strings.Builder
	b.WriteString("## User context:\n")
	b.WriteString("Timezone: ")
	b.WriteString(tz)
	b.WriteString("\n\n")
	b.WriteString("## Subject:\n")
	b.WriteString(c.Subject)
	b.WriteString("\n\n## Previous results (JSON):\n")
	b.WriteString(c.PreviousResultsString())

	if !c.Schema.Zero() {
		schemaJSON, err := json.Marshal(c.Schema)
		if err == nil {
			b.WriteString("\n## Monitor schema:\n")
			b.Write(schemaJSON)
			b.WriteString("\n")
		}
	}

	return b.String()
}

func (c CheckParams) PreviousResultsString() string {
	var prevs strings.Builder
	for _, pr := range c.PreviousResults {
		prevs.WriteString(must.NoErrVal(pr.PromptJSON()))
		prevs.WriteString("\n")
	}
	return prevs.String()
}

func sanitizeXAIOutput(in string) string {
	return strings.ReplaceAll(in, "</xai:function_call>", "")
}
