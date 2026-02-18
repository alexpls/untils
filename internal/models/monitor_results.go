package models

import (
	"encoding/json"
	"fmt"
	"strings"
)

func (mr MonitorResult) PromptJSON() (string, error) {
	payload := struct {
		Headline         string            `json:"headline"`
		Subtitle         string            `json:"subtitle"`
		Data             MonitorUpdateData `json:"data"`
		LatestCheckRanAt string            `json:"latest_check_ran_at"`
		UserFeedback     string            `json:"user_feedback,omitempty"`
		SourcesUsed      []string          `json:"sources_used,omitempty"`
	}{
		Headline:         mr.Headline,
		Subtitle:         mr.Subtitle,
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

func (mr MonitorResult) RenderHeadline(
	renderer MonitorFieldsRenderer,
	renderCtx MonitorFieldsRenderContext,
) (string, error) {
	return mr.Data.Fields.RenderTemplate(mr.Headline, renderer, renderCtx)
}

func (mr MonitorResult) RenderSubtitle(
	renderer MonitorFieldsRenderer,
	renderCtx MonitorFieldsRenderContext,
) (string, error) {
	if strings.TrimSpace(mr.Subtitle) == "" {
		return "", nil
	}
	return mr.Data.Fields.RenderTemplate(mr.Subtitle, renderer, renderCtx)
}

func (mr MonitorResult) MustRenderHeadline(
	renderer MonitorFieldsRenderer,
	renderCtx MonitorFieldsRenderContext,
) string {
	rendered, err := mr.RenderHeadline(renderer, renderCtx)
	if err != nil {
		panic(err)
	}
	return rendered
}

func (mr MonitorResult) MustRenderSubtitle(
	renderer MonitorFieldsRenderer,
	renderCtx MonitorFieldsRenderContext,
) string {
	rendered, err := mr.RenderSubtitle(renderer, renderCtx)
	if err != nil {
		panic(err)
	}
	return rendered
}
