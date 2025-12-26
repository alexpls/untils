package llm

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
)

type expertDefault struct {
	service *Service
}

func newExpertDefault(service *Service) expert {
	return &expertDefault{service: service}
}

//go:embed expert_default_prompt.md
var expertDefaultPrompt string

func (e *expertDefault) performCheck(ctx context.Context, params *CheckParams) (*CheckResult, error) {
	var previousResponses strings.Builder
	for _, pr := range params.PreviousResults {
		d, err := json.Marshal(pr)
		if err != nil {
			e.service.logger.Error("error unmarshaling previous response", "error", err)
		} else {
			previousResponses.Write(d)
		}
	}

	messages := []responses.ResponseInputItemUnionParam{
		systemMessage(expertDefaultPrompt),
		userMessage("Subject: " + params.Subject +
			"\n\nInstructions: " + params.Instructions +
			"\n\nPrevious result: \n" + previousResponses.String()),
	}

	for {
		resp, err := e.service.response(ctx, responses.ResponseNewParams{
			Model: model,
			Input: inputItems(messages...),
			Text:  jsonSchemaResponse(CheckResult{}),
			Tools: append(webSearchTools(), browserTools()...),
		})
		if err != nil {
			return nil, err
		}

		if len(resp.toolCalls) > 0 {
			for _, item := range resp.toolCalls {
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

			continue
		}

		sanitized := sanitizeXAIOutput(resp.OutputText())
		res := CheckResult{}
		if err := json.Unmarshal([]byte(sanitized), &res); err != nil {
			return nil, fmt.Errorf("unmarshaling llm response: %w", err)
		}

		return &res, nil
	}
}
