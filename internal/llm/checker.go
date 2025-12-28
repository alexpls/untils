package llm

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
)

type checker struct {
	service  *Service
	messages []responses.ResponseInputItemUnionParam
}

func newChecker(service *Service) *checker {
	return &checker{service: service}
}

//go:embed checker_prompt.md
var checkerPrompt string

func (c *checker) perform(ctx context.Context, params *CheckParams) (*CheckResult, error) {
	var err error

	prevs, err := params.PreviousResultsString()
	if err != nil {
		return nil, err
	}

	c.messages = []responses.ResponseInputItemUnionParam{
		systemMessage(checkerPrompt),
		userMessage("Subject: " + params.Subject +
			"\n\nInstructions: " + params.Instructions +
			"\n\nPrevious results: \n" + prevs),
	}

	var resp *responseResult
	maxTurns := 10
	turn := 0

	for {
		if turn >= maxTurns {
			return nil, fmt.Errorf("exceeded max turns: %w", err)
		}
		turn++

		resp, err = c.service.response(ctx, responses.ResponseNewParams{
			Model: model,
			Input: inputItems(c.messages...),
			Text:  jsonSchemaResponse(CheckResult{}),
			Tools: append(browserTools(), searchTools()...),
		})

		if len(resp.toolCalls) > 0 {
			c.handleToolCalls(ctx, resp.toolCalls)
			continue
		}

		sanitized := sanitizeXAIOutput(resp.OutputText())
		res := &CheckResult{}
		if err := json.Unmarshal([]byte(sanitized), res); err != nil {
			c.messages = append(c.messages, systemMessage("error: invalid response format"))
			continue
		}

		return res, nil
	}
}

func (c *checker) handleToolCalls(ctx context.Context, toolCalls []responses.ResponseFunctionToolCall) {
	for _, call := range toolCalls {
		params, err := toolCallParams(call.Name, call.Arguments)
		if err != nil {
			c.messages = append(c.messages, systemMessage("error: invalid tool call"))
			continue
		}

		result, err := c.service.handleToolCall(ctx, call.Name, params)
		if err != nil {
			c.messages = append(c.messages, systemMessage("error executing tool call: "+err.Error()))
			continue
		}

		c.messages = append(c.messages, responses.ResponseInputItemUnionParam{
			OfFunctionCallOutput: &responses.ResponseInputItemFunctionCallOutputParam{
				CallID: call.CallID,
				Output: responses.ResponseInputItemFunctionCallOutputOutputUnionParam{
					OfString: openai.String(result),
				},
			},
		})
	}
}
