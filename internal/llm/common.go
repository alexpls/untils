package llm

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Date struct {
	Date          string `json:"date"`
	PastTenseVerb string `json:"past_tense_verb"`
}

type Citations []Citation

type Citation struct {
	URL          string `json:"url"`
	WebsiteTitle string `json:"website_title"`
	PageTitle    string `json:"page_title"`
}

type CheckParams struct {
	Subject         string
	Instructions    string
	PreviousResults []CheckResult
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

type CheckResult struct {
	Success             bool      `json:"success"`
	Reason              string    `json:"reason"`
	DifferentToPrevious bool      `json:"different_to_previous"`
	ResultPlaintext     string    `json:"result_plaintext"`
	Date                Date      `json:"date"`
	Citations           Citations `json:"citations"` // TODO: change to Sources
}

func sanitizeXAIOutput(in string) string {
	return strings.ReplaceAll(in, "</xai:function_call>", "")
}
