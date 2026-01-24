package llm

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/alexpls/untils/internal/logging"
	"github.com/alexpls/untils/internal/search"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
)

var modelNonReasoning = "grok-4-1-fast-non-reasoning"
var modelReasoning = "grok-4-1-fast-reasoning"

type Service struct {
	client      *openai.Client
	logger      *slog.Logger
	webSearcher search.WebSearcher
}

func NewService(client *openai.Client, logger *slog.Logger, webSearcher search.WebSearcher) *Service {
	return &Service{
		client:      client,
		logger:      logger,
		webSearcher: webSearcher,
	}
}

type responseResult struct {
	*responses.Response
	toolCalls []responses.ResponseFunctionToolCall
}

func (s *Service) response(ctx context.Context, params responses.ResponseNewParams) (*responseResult, error) {
	llmEvent, _ := logging.GetOrCreateFromContext(ctx, newLLMEvent)
	turn := llmEvent.newTurn()

	defer turn.finish()

	resp, err := s.client.Responses.New(ctx, params)

	if err != nil {
		turn.addError(err)
		return nil, fmt.Errorf("fetching llm response: %w", err)
	}

	cost, err := calculateCost(modelNonReasoning, resp)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to calculate cost", "error", err)
	} else {
		turn.addCost(cost)
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
