package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/alexpls/untils/internal/openai"
)

type openAIProvider struct {
	client *openai.Client
}

func NewOpenAIProvider(client *openai.Client) Provider {
	return &openAIProvider{client: client}
}

func (p *openAIProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	params := openai.CreateRequest{
		Model: req.Model,
		Input: toOpenAIInput(req.Messages),
	}

	if req.ResponseSchema != nil {
		params.Text = &openai.TextConfig{
			Format: &openai.JSONSchemaFormat{
				Name:   req.ResponseName,
				Strict: true,
				Schema: req.ResponseSchema,
			},
		}
	}

	if len(req.Tools) > 0 {
		params.Tools = toOpenAITools(req.Tools)
		parallel := req.ParallelToolCalls
		params.ParallelToolCalls = &parallel
	}

	resp, err := p.client.Responses.New(ctx, params)
	if err != nil {
		err = fmt.Errorf("fetching llm response: %w", err)
		if openAIErrorIsNonRetryable(err) {
			return nil, nonRetryableProviderErr(err)
		}
		return nil, err
	}

	calls := resp.FunctionCalls()
	toolCalls := make([]ToolCall, 0, len(calls))
	for _, call := range calls {
		toolCalls = append(toolCalls, ToolCall{
			ID:        call.CallID,
			Name:      call.Name,
			Arguments: call.Arguments,
		})
	}

	return &CompletionResponse{
		Raw:       json.RawMessage(resp.RawJSON()),
		Output:    resp.OutputText(),
		ToolCalls: toolCalls,
		Usage: TokenUsage{
			InputTokens:  resp.Usage.InputTokens,
			OutputTokens: resp.Usage.OutputTokens,
			ExtraFields:  resp.Usage.ExtraFields,
		},
	}, nil
}

func openAIErrorIsNonRetryable(err error) bool {
	var apiErr *openai.Error
	if !errors.As(err, &apiErr) {
		return false
	}

	switch apiErr.StatusCode {
	case http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusNotFound:
		return true
	case http.StatusTooManyRequests:
		return openAIErrorIsQuotaExhausted(apiErr)
	default:
		return false
	}
}

func openAIErrorIsQuotaExhausted(err *openai.Error) bool {
	message := strings.ToLower(err.Message + " " + err.RawJSON())
	return strings.Contains(message, "used all available credits") ||
		strings.Contains(message, "monthly spending limit") ||
		strings.Contains(message, "purchase more credits") ||
		strings.Contains(message, "insufficient_quota")
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
		if strings.HasPrefix(model, "gpt-") {
			return 0.0, nil
		}
		return 0.0, fmt.Errorf("unsupported model: %s", model)
	}

	return cost, nil
}

func toOpenAIInput(messages []Message) []openai.InputItem {
	out := make([]openai.InputItem, 0, len(messages))
	for _, msg := range messages {
		switch {
		case msg.ToolCall != nil:
			out = append(out, openai.InputItem{
				FunctionCall: &openai.InputFunctionCall{
					CallID:    msg.ToolCall.ID,
					Name:      msg.ToolCall.Name,
					Arguments: msg.ToolCall.Arguments,
				},
			})
		case msg.ToolOutput != nil:
			out = append(out, openai.InputItem{
				FunctionCallOut: &openai.InputFunctionCallOutput{
					CallID: msg.ToolOutput.CallID,
					Output: msg.ToolOutput.Output,
				},
			})
		default:
			out = append(out, openai.InputItem{
				Message: &openai.InputMessage{
					Role:    openai.MessageRole(msg.Role),
					Content: msg.Content,
				},
			})
		}
	}
	return out
}

func toOpenAITools(tools []ToolDefinition) []openai.Tool {
	out := make([]openai.Tool, 0, len(tools))
	for _, tool := range tools {
		out = append(out, openai.Tool{
			Name:        tool.Name,
			Description: tool.Description,
			Parameters:  tool.Parameters,
		})
	}
	return out
}
