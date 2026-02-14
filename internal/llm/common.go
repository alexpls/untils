package llm

import (
	"strings"

	"github.com/alexpls/untils/internal/models"
)

type CheckParams struct {
	UserID          int64
	MonitorID       int64
	MonitorCheckID  int64
	Subject         string
	PreviousResults []*models.GetPreviousResultsWithCheckRow
}

func (c CheckParams) UserMessageString() string {
	return "## Subject:\n" + c.Subject + "\n\n## Previous results: \n" + c.PreviousResultsString()
}

func (c CheckParams) PreviousResultsString() string {
	var prevs strings.Builder
	for _, pr := range c.PreviousResults {
		prevs.WriteString(pr.MonitorResultsWithLatestCheck.Markdown())
		prevs.WriteString("\n")
	}
	return prevs.String()
}

func sanitizeXAIOutput(in string) string {
	return strings.ReplaceAll(in, "</xai:function_call>", "")
}
