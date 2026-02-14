package llm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
)

type openAIProvider struct {
	client *openai.Client
}

func NewOpenAIProvider(client *openai.Client) Provider {
	return &openAIProvider{client: client}
}

func (p *openAIProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	params := responses.ResponseNewParams{
		Model: req.Model,
		Input: responses.ResponseNewParamsInputUnion{
			OfInputItemList: toOpenAIInput(req.Messages),
		},
	}

	if req.ResponseSchema != nil {
		params.Text = responses.ResponseTextConfigParam{
			Format: responses.ResponseFormatTextConfigUnionParam{
				OfJSONSchema: &responses.ResponseFormatTextJSONSchemaConfigParam{
					Name:   req.ResponseName,
					Strict: openai.Bool(true),
					Schema: req.ResponseSchema,
				},
			},
		}
	}

	if len(req.Tools) > 0 {
		params.Tools = toOpenAITools(req.Tools)
		params.ParallelToolCalls = openai.Bool(req.ParallelToolCalls)
	}

	resp, err := p.client.Responses.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("fetching llm response: %w", err)
	}

	toolCalls := make([]ToolCall, 0)
	for _, item := range resp.Output {
		switch item.AsAny().(type) {
		case responses.ResponseFunctionToolCall:
			call := item.AsFunctionCall()
			toolCalls = append(toolCalls, ToolCall{
				ID:        call.CallID,
				Name:      call.Name,
				Arguments: call.Arguments,
			})
		}
	}

	extraFields := make(map[string]json.RawMessage, len(resp.Usage.JSON.ExtraFields))
	for k, v := range resp.Usage.JSON.ExtraFields {
		extraFields[k] = json.RawMessage(v.Raw())
	}

	return &CompletionResponse{
		Raw:       json.RawMessage(resp.RawJSON()),
		Output:    resp.OutputText(),
		ToolCalls: toolCalls,
		Usage: TokenUsage{
			InputTokens:  int64(resp.Usage.InputTokens),
			OutputTokens: int64(resp.Usage.OutputTokens),
			ExtraFields:  extraFields,
		},
	}, nil
}

func (p *openAIProvider) CalculateCostUSD(model string, usage TokenUsage) (float64, error) {
	cost := 0.0

	switch model {
	case "grok-4-1-fast-non-reasoning", "grok-4-1-fast-reasoning":
		// https://docs.x.ai/docs/models
		per1MInputToken, per1MOutputToken := 0.2, 0.5
		per1KSearchCalls, per1KXSearchCalls := 5.0, 5.0

		cost += float64(usage.InputTokens) / 1_000_000 * per1MInputToken
		cost += float64(usage.OutputTokens) / 1_000_000 * per1MOutputToken
		if raw, ok := usage.ExtraFields["server_side_tool_usage_details"]; ok {
			toolUsage := struct {
				WebSearchCalls int `json:"web_search_calls"`
				XSearchCalls   int `json:"x_search_calls"`
			}{}
			if err := json.Unmarshal(raw, &toolUsage); err != nil {
				return 0.0, fmt.Errorf("unmarshaling tool usage: %w", err)
			}
			cost += float64(toolUsage.WebSearchCalls) / 1_000 * per1KSearchCalls
			cost += float64(toolUsage.XSearchCalls) / 1_000 * per1KXSearchCalls
		}
	default:
		return 0.0, fmt.Errorf("unsupported model: %s", model)
	}

	return cost, nil
}

func toOpenAIInput(messages []Message) responses.ResponseInputParam {
	out := make([]responses.ResponseInputItemUnionParam, 0, len(messages))
	for _, msg := range messages {
		switch {
		case msg.ToolCall != nil:
			out = append(out, responses.ResponseInputItemUnionParam{
				OfFunctionCall: &responses.ResponseFunctionToolCallParam{
					CallID:    msg.ToolCall.ID,
					Name:      msg.ToolCall.Name,
					Arguments: msg.ToolCall.Arguments,
				},
			})
		case msg.ToolOutput != nil:
			out = append(out, responses.ResponseInputItemUnionParam{
				OfFunctionCallOutput: &responses.ResponseInputItemFunctionCallOutputParam{
					CallID: msg.ToolOutput.CallID,
					Output: responses.ResponseInputItemFunctionCallOutputOutputUnionParam{
						OfString: openai.String(msg.ToolOutput.Output),
					},
				},
			})
		default:
			out = append(out, responses.ResponseInputItemUnionParam{
				OfMessage: &responses.EasyInputMessageParam{
					Content: responses.EasyInputMessageContentUnionParam{
						OfString: openai.String(msg.Content),
					},
					Role: responses.EasyInputMessageRole(msg.Role),
				},
			})
		}
	}
	return out
}

func toOpenAITools(tools []ToolDefinition) []responses.ToolUnionParam {
	out := make([]responses.ToolUnionParam, 0, len(tools))
	for _, tool := range tools {
		out = append(out, responses.ToolUnionParam{
			OfFunction: &responses.FunctionToolParam{
				Name:        tool.Name,
				Description: openai.String(tool.Description),
				Parameters:  tool.Parameters,
			},
		})
	}
	return out
}
