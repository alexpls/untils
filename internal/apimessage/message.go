package apimessage

import (
	"fmt"

	"github.com/alexpls/untils/internal/models"
	"github.com/alexpls/untils/internal/monitorfieldrenderers"
)

func NewErrorResponse(code, message string) ErrorResponse {
	return ErrorResponse{
		Data: nil,
		Error: APIError{
			Code:    code,
			Message: message,
		},
	}
}

func NewListLatestResultsResponse(results []ResultSummary) ListLatestResultsResponse {
	return ListLatestResultsResponse{
		Error: nil,
		Data: struct {
			ResultSummaries []ResultSummary `json:"result_summaries"`
			Type            string          `json:"type"`
		}{
			Type:            "result_summaries",
			ResultSummaries: results,
		},
	}
}

func NewGetMonitorResponse(monitor Monitor) GetMonitorResponse {
	return GetMonitorResponse{
		Error: nil,
		Data: struct {
			Monitor Monitor `json:"monitor"`
			Type    string  `json:"type"`
		}{
			Type:    "monitor",
			Monitor: monitor,
		},
	}
}

func NewListResultsResponse(results []Result, links PaginationLinks) ListResultsResponse {
	return ListResultsResponse{
		Error: nil,
		Data: struct {
			Links   PaginationLinks `json:"links"`
			Results []Result        `json:"results"`
			Type    string          `json:"type"`
		}{
			Type:    "results",
			Results: results,
			Links:   links,
		},
	}
}

func BuildMonitorMessage(monitor models.Monitor) Monitor {
	return Monitor{
		Type:      "monitor",
		ID:        monitor.ID,
		CreatedAt: monitor.CreatedAt,
		Status:    MonitorStatus(monitor.Status),
		Subject:   monitor.Subject.String,
	}
}

func BuildResultMessage(result models.MonitorResult) (Result, error) {
	renderer := monitorfieldrenderers.TextRenderer{}
	renderCtx := models.MonitorFieldsRenderContext{}

	fields := make([]ResultField, len(result.Data.Fields))
	for i, f := range result.Data.Fields {
		fields[i] = ResultField{
			Type:  "result_field",
			Name:  f.Name,
			Value: f.Value,
		}
	}

	headline, err := result.RenderHeadline(renderer, renderCtx)
	if err != nil {
		return Result{}, fmt.Errorf("rendering headline for result %d: %w", result.ID, err)
	}
	subtitle, err := result.RenderSubtitle(renderer, renderCtx)
	if err != nil {
		return Result{}, fmt.Errorf("rendering subtitle for result %d: %w", result.ID, err)
	}

	resultMessage := Result{
		Type:     "result",
		ID:       result.ID,
		Hidden:   result.Hidden,
		Headline: headline,
		Subtitle: subtitle,
		Fields:   fields,
	}
	if result.Correction.Valid {
		resultMessage.Correction = &result.Correction.String
	}
	return resultMessage, nil
}

func BuildResultSummaryMessage(monitor models.Monitor, result models.MonitorResult) (ResultSummary, error) {
	resultMessage, err := BuildResultMessage(result)
	if err != nil {
		return ResultSummary{}, err
	}
	return ResultSummary{
		Type:           "result_summary",
		ID:             resultMessage.ID,
		MonitorID:      monitor.ID,
		MonitorSubject: monitor.Subject.String,
		CreatedAt:      result.CreatedAt,
		Headline:       resultMessage.Headline,
		Subtitle:       resultMessage.Subtitle,
		Fields:         resultMessage.Fields,
	}, nil
}
