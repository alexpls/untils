package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/openai/openai-go/v3/responses"
)

type ValidateMonitorPromptInput struct {
	Subject string
}

type ValidateMonitorPromptResponse struct {
	Approved       bool   `json:"approved"`
	RejectedReason string `json:"rejected_reason"`
}

var validatePromptInstructions = preamble + `
## Instructions

- Validate that the subject they wish to monitor is suitable for monitoring.
- If it's not suitable, then you must reject the subject and offer a concise
  reason explaining why. This will be displayed directly to the user on the
  application's UI.

## Subject suitability rules

- It must be possible to find the result for the subject on the public internet.
- The subject being monitored may not be anything illegal or that would
  otherwise cause harm to a human.
- The subject can be something that changes often, but within reason. For example
  if the user asks "what time is it?" then every time the monitor is checked
  a different result will show up, which defeats the purpose of providing meaningful
  updates when something changes.

## Output rules

- If you have approved the subject, then return an empty string in the "rejected_reason" field
`

func (s *Service) ValidateMonitorPrompt(ctx context.Context, input ValidateMonitorPromptInput) (*ValidateMonitorPromptResponse, error) {
	start := time.Now()

	resp, err := s.client.Responses.New(ctx, responses.ResponseNewParams{
		Model: model,
		Input: responseInput(
			validatePromptInstructions,
			fmt.Sprintf("Monitor subject to validate: %s", input.Subject),
		),
		Text: jsonSchemaResponse("validate_monitor_subject", ValidateMonitorPromptResponse{}),
	})

	s.logger.Info("validate monitor generated response",
		"model", model,
		"duration_ms", time.Since(start).Milliseconds(),
		"success", err == nil)

	if err != nil {
		return nil, fmt.Errorf("fetching llm response: %w", err)
	}

	sanitized := sanitizeXAIOutput(resp.OutputText())

	res := ValidateMonitorPromptResponse{}
	if err = json.Unmarshal([]byte(sanitized), &res); err != nil {
		return nil, fmt.Errorf("parsing llm output to json: %w", err)
	}

	return &res, nil
}
