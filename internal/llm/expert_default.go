package llm

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/openai/openai-go/v3"
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
	messages := []responses.ResponseInputItemUnionParam{
		systemMessage(expertDefaultPrompt),
		userMessage("Subject: " + params.Subject +
			"\n\nInstructions: " + params.Instructions +
			"\n\nPrevious result: " + params.PreviousResult),
	}

	for {
		resp, err := e.service.response(ctx, responses.ResponseNewParams{
			Model: model,
			Input: inputItems(messages...),
			Text:  jsonSchemaResponse(CheckResponse{}),
			Tools: append(webSearchTools(), browserTools()...),
		})
		if err != nil {
			return nil, err
		}

		calledTools := false

		for _, item := range resp.Output {
			switch item.AsAny().(type) {
			case responses.ResponseFunctionToolCall:
				calledTools = true
				item := item.AsFunctionCall()
				res, err := handleToolCall(ctx, item.Name, item.Arguments)
				if err != nil {
					e.service.logger.Error("error handling tool call", "error", err)
				}
				messages = append(messages, responses.ResponseInputItemUnionParam{
					OfFunctionCallOutput: &responses.ResponseInputItemFunctionCallOutputParam{
						CallID: item.CallID,
						Output: responses.ResponseInputItemFunctionCallOutputOutputUnionParam{
							OfString: openai.String(res),
						},
					},
				})
			}
		}

		if calledTools {
			continue // go again
		}

		sanitized := sanitizeXAIOutput(resp.OutputText())
		res := CheckResponse{}
		if err := json.Unmarshal([]byte(sanitized), &res); err != nil {
			return nil, fmt.Errorf("unmarshaling llm response: %w", err)
		}

		return &res, nil
	}
}
