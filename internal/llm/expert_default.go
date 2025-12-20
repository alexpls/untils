package llm

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/openai/openai-go/v3/responses"
)

type ExpertDefault struct {
	service *Service
}

func NewExpertDefault(service *Service) Expert {
	return &ExpertDefault{service: service}
}

//go:embed expert_default_prompt.md
var expertDefaultPrompt string

func (e *ExpertDefault) PerformCheck(ctx context.Context, params *CheckParams) (*CheckResponse, error) {
	messages := inputItems(
		systemMessage(expertDefaultPrompt),
		userMessage("Subject: "+params.Subject+
			"\n\nInstructions: "+params.Instructions+
			"\n\nPrevious result: "+params.PreviousResult),
	)

	resp, err := e.service.response(ctx, responses.ResponseNewParams{
		Model: model,
		Input: messages,
		Text:  jsonSchemaResponse(CheckResponse{}),
		Tools: webSearchTool(),
	})
	if err != nil {
		return nil, err
	}

	sanitized := sanitizeXAIOutput(resp.OutputText())
	res := CheckResponse{}
	if err := json.Unmarshal([]byte(sanitized), &res); err != nil {
		return nil, fmt.Errorf("unmarshaling llm response: %w", err)
	}

	return &res, nil
}
