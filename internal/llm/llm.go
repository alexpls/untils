package llm

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
)

var model = "grok-4-1-fast-non-reasoning"

type Service struct {
	client *openai.Client
	logger *slog.Logger
}

func NewService(client *openai.Client, logger *slog.Logger) *Service {
	return &Service{
		client: client,
		logger: logger,
	}
}

type responseResult struct {
	*responses.Response
	toolCalls []responses.ResponseFunctionToolCall
}

func (s *Service) response(ctx context.Context, params responses.ResponseNewParams) (*responseResult, error) {
	stats := statsFromContext(ctx)
	turn := stats.newTurn()

	resp, err := s.client.Responses.New(ctx, params)

	turn.end = time.Now()

	s.logger.Debug("generated response",
		"model", model,
		"turn_duration_ms", turn.duration().Milliseconds(),
		"turn_num", len(stats.turns),
		"success", err == nil)

	if err != nil {
		turn.err = err
		return nil, fmt.Errorf("fetching llm response: %w", err)
	}

	cost, err := calculateCost(model, resp)
	if err != nil {
		s.logger.Error("failed to calculate cost", "error", err)
	} else {
		turn.cost = cost
		s.logger.Debug("calculated cost", "cost_usd", cost, "total_cost", stats.totalCost())
	}

	toolCalls := extractToolCalls(resp.Output)
	for _, item := range toolCalls {
		turn.incrToolCall(item.Name)
	}

	return &responseResult{
		toolCalls: toolCalls,
		Response:  resp,
	}, nil
}

func userMessage(content string) responses.ResponseInputItemUnionParam {
	return responses.ResponseInputItemUnionParam{
		OfMessage: &responses.EasyInputMessageParam{
			Content: responses.EasyInputMessageContentUnionParam{
				OfString: openai.String(content),
			},
			Role: "user",
		},
	}
}

func systemMessage(content string) responses.ResponseInputItemUnionParam {
	return responses.ResponseInputItemUnionParam{
		OfMessage: &responses.EasyInputMessageParam{
			Content: responses.EasyInputMessageContentUnionParam{
				OfString: openai.String(content),
			},
			Role: "system",
		},
	}
}

func inputItems(messages ...responses.ResponseInputItemUnionParam) responses.ResponseNewParamsInputUnion {
	return responses.ResponseNewParamsInputUnion{
		OfInputItemList: responses.ResponseInputParam(messages),
	}
}

func webSearchTools() []responses.ToolUnionParam {
	return []responses.ToolUnionParam{
		responses.ToolParamOfWebSearch("web_search"),
		responses.ToolParamOfWebSearch("x_search"),
	}
}

func extractToolCalls(outputs []responses.ResponseOutputItemUnion) (out []responses.ResponseFunctionToolCall) {
	for _, item := range outputs {
		switch item.AsAny().(type) {
		case responses.ResponseFunctionToolCall:
			item := item.AsFunctionCall()
			out = append(out, item)
		}
	}
	return out
}
