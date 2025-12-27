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

	var logAttrs []any

	defer func() {
		logAttrs = append(logAttrs,
			slog.String("model", model),
			slog.Int("turn_num", len(stats.turns)),
		)
		s.logger.Debug("turn complete", logAttrs...)
	}()

	resp, err := s.client.Responses.New(ctx, params)

	turn.end = time.Now()

	logAttrs = append(logAttrs,
		slog.Duration("duration", turn.duration()),
		slog.Bool("success", err == nil),
	)

	if err != nil {
		turn.err = err
		return nil, fmt.Errorf("fetching llm response: %w", err)
	}

	logAttrs = append(logAttrs, slog.String("response", resp.OutputText()))
	// logAttrs = append(logAttrs, slog.String("raw_json", resp.RawJSON()))

	cost, err := calculateCost(model, resp)
	if err != nil {
		s.logger.Error("failed to calculate cost", "error", err)
	} else {
		turn.cost = cost
		logAttrs = append(logAttrs,
			slog.Float64("cost_usd", cost),
			slog.Int64("input_tokens", resp.Usage.InputTokens),
			slog.Int64("output_tokens", resp.Usage.OutputTokens),
		)
	}

	toolCalls := extractToolCalls(resp.Output)
	for _, item := range toolCalls {
		logAttrs = append(logAttrs, slog.String("tool."+item.Name, item.Arguments))
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
