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

func (s *Service) response(ctx context.Context, params responses.ResponseNewParams) (*responses.Response, error) {
	start := time.Now()

	resp, err := s.client.Responses.New(ctx, params)

	s.logger.Info("generated response",
		"model", model,
		"duration_ms", time.Since(start).Milliseconds(),
		"success", err == nil)

	if err != nil {
		return nil, fmt.Errorf("fetching llm response: %w", err)
	}

	cost, err := calculateCost(model, resp)
	if err != nil {
		s.logger.Error("failed to calculate cost", "error", err)
	} else {
		s.logger.Info("calculated cost", "cost_usd", cost)
	}

	return resp, nil
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

func webSearchTool() []responses.ToolUnionParam {
	return []responses.ToolUnionParam{
		responses.ToolParamOfWebSearch("web_search"),
		responses.ToolParamOfWebSearch("x_search"),
	}
}
