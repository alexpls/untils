package llm

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/alexpls/untils/internal/db/models"
)

type CheckParams struct {
	Subject         string
	Instructions    string
	PreviousResults []models.CheckResult
}

func (c CheckParams) PreviousResultsString() (string, error) {
	var prevs strings.Builder
	for _, pr := range c.PreviousResults {
		d, err := json.Marshal(pr)
		if err != nil {
			return "", fmt.Errorf("marshaling previous results: %w", err)
		} else {
			prevs.Write(d)
			prevs.WriteString("\n")
		}
	}
	return prevs.String(), nil
}

func sanitizeXAIOutput(in string) string {
	return strings.ReplaceAll(in, "</xai:function_call>", "")
}
