package models

import (
	"encoding/json"
	"fmt"
)

func (mr MonitorResult) PromptJSON(schema MonitorSchemaData) (string, error) {
	payload := struct {
		Schema           MonitorSchemaData `json:"schema"`
		Data             MonitorUpdateData `json:"data"`
		LatestCheckRanAt string            `json:"latest_check_ran_at"`
		UserFeedback     string            `json:"user_feedback,omitempty"`
		SourcesUsed      []string          `json:"sources_used,omitempty"`
	}{
		Schema:           schema,
		Data:             mr.Data,
		LatestCheckRanAt: mr.LastConfirmedAt.Format("January 2, 2006 at 3:04 PM"),
	}

	if mr.Feedback.Valid {
		payload.UserFeedback = mr.Feedback.String
	}

	if mr.Citations != nil && len(*mr.Citations) > 0 {
		payload.SourcesUsed = make([]string, 0, len(*mr.Citations))
		for _, citation := range *mr.Citations {
			payload.SourcesUsed = append(payload.SourcesUsed, citation.URL)
		}
	}

	b, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshaling json: %w", err)
	}

	return string(b), nil
}
