package models

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

func (mr MonitorResult) PromptJSON() (string, error) {
	return monitorResultPromptJSON(mr, mr.CreatedAt)
}

func (mrwc GetPreviousResultsWithCheckRow) PromptJSON() (string, error) {
	latestCheckRanAt := mrwc.MonitorResult.CreatedAt
	if mrwc.MonitorCheck.DoneAt != nil {
		latestCheckRanAt = *mrwc.MonitorCheck.DoneAt
	}

	return monitorResultPromptJSON(mrwc.MonitorResult, latestCheckRanAt)
}

func (mr MonitorResult) IsVisible() bool {
	return !mr.Hidden
}

func (mr MonitorResult) CanApplyCorrection(latestVisible *MonitorResult) bool {
	return latestVisible != nil && mr.ID == latestVisible.ID
}

func LatestVisiblePreviousResult(previousResults []*GetPreviousResultsWithCheckRow) *GetPreviousResultsWithCheckRow {
	for _, result := range previousResults {
		if result.MonitorResult.IsVisible() {
			return result
		}
	}

	return nil
}

func monitorResultPromptJSON(mr MonitorResult, latestCheckRanAt time.Time) (string, error) {
	payload := struct {
		Headline         string            `json:"headline"`
		Subtitle         string            `json:"subtitle"`
		Data             MonitorUpdateData `json:"data"`
		LatestCheckRanAt string            `json:"latest_check_ran_at"`
		Correction       string            `json:"correction,omitempty"`
		HiddenInUI       bool              `json:"hidden_in_ui,omitempty"`
		SourcesUsed      []string          `json:"sources_used,omitempty"`
	}{
		Headline:         mr.Headline,
		Subtitle:         mr.Subtitle,
		Data:             mr.Data,
		LatestCheckRanAt: latestCheckRanAt.Format("January 2, 2006 at 3:04 PM"),
	}

	if mr.Correction.Valid {
		payload.Correction = mr.Correction.String
	}

	if mr.Hidden {
		payload.HiddenInUI = true
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
