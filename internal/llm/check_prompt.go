package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/openai/openai-go/v3/responses"
)

type CheckPromptParams struct {
	Subject        string
	Instructions   string
	PreviousResult string
}

type CheckPromptResponse struct {
	DifferentToPrevious bool      `json:"different_to_previous"`
	ResponsePlaintext   string    `json:"response_plaintext"`
	Date                Date      `json:"date"`
	Citations           Citations `json:"citations"`
}

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

var checkPromptInstructions = preamble + checkInstructions + `
## Previous result

%s
`

func (s *Service) CheckPrompt(ctx context.Context, input CheckPromptParams) (*CheckPromptResponse, error) {
	start := time.Now()

	in := responseInput(
		fmt.Sprintf(checkPromptInstructions, input.PreviousResult),
		fmt.Sprintf("Subject to check: %s\bUser specified instructions: %s", input.Subject, input.Instructions),
	)

	resp, err := s.client.Responses.New(ctx, responses.ResponseNewParams{
		Model:     model,
		Input:     in,
		Reasoning: reasoning,
		Tools:     webSearchTool(),
		Text:      jsonSchemaResponse("check_response", CheckPromptResponse{}),
	})

	s.logger.Info("generated response",
		"model", model,
		"duration_ms", time.Since(start).Milliseconds(),
		"usage_json", resp.Usage.RawJSON(),
		"success", err == nil)

	s.logger.Debug("llm response details",
		"input", in,
		"usage_json", resp.Usage.RawJSON(),
		"raw_response", resp.OutputText())

	if err != nil {
		return nil, fmt.Errorf("fetching llm response: %w", err)
	}

	sanitized := sanitizeXAIOutput(resp.OutputText())

	res := CheckPromptResponse{}
	if err = json.Unmarshal([]byte(sanitized), &res); err != nil {
		return nil, fmt.Errorf("parsing llm output to json: %w", err)
	}

	return &res, nil
}
