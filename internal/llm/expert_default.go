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
	var prevs strings.Builder
	for _, pr := range params.PreviousResults {
		d, err := json.Marshal(pr)
		if err != nil {
			return nil, fmt.Errorf("marshaling previous result: %w", err)
		} else {
			prevs.Write(d)
			prevs.WriteString("\n")
		}
	}

	var sources strings.Builder
	for _, src := range params.Sources {
		s, err := json.Marshal(src)
		if err != nil {
			return nil, fmt.Errorf("marshaling source: %w", err)
		} else {
			sources.Write(s)
			sources.WriteString("\n")
		}
	}

	messages := []responses.ResponseInputItemUnionParam{
		systemMessage(expertDefaultPrompt),
		userMessage("Subject: " + params.Subject +
			"\n\nInstructions: " + params.Instructions +
			"\n\nPrevious results: \n" + prevs.String() +
			"\n\nSources: \n" + sources.String()),
	}

	var resp *responseResult
	var err error
	maxTurns := 10
	turn := 0

	for {
		if turn >= maxTurns {
			return nil, fmt.Errorf("exceeded max turns: %w", err)
		}
		turn++

		resp, err = e.service.response(ctx, responses.ResponseNewParams{
			Model: model,
			Input: inputItems(messages...),
			Text:  jsonSchemaResponse(CheckResult{}),
			Tools: browserTools(),
		})
		if err != nil {
			return nil, err
		}

		if len(resp.toolCalls) > 0 {
			for _, item := range resp.toolCalls {
				var res string
				res, err = handleToolCall(ctx, item.Name, item.Arguments)
				if err != nil {
					err = fmt.Errorf("handling tool call %q: %w", item.Name, err)
					e.service.logger.Error("error handling tool call", "name", item.Name, "error", err)
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
		if err = json.Unmarshal([]byte(sanitized), &res); err != nil {
			return nil, fmt.Errorf("unmarshaling llm response: %w", err)
		}

		return &res, nil
	}
}
