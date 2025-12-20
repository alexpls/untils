package llm

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
	"github.com/openai/openai-go/v3/shared"
)

var model = "grok-4-1-fast-non-reasoning"
var reasoning = shared.ReasoningParam{}

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

	if err != nil {
		return nil, fmt.Errorf("fetching llm response: %w", err)
	}

	s.logger.Info("generated response",
		"model", model,
		"duration_ms", time.Since(start).Milliseconds(),
		"usage_json", resp.Usage.RawJSON(),
		"success", err == nil)

	s.logger.Debug("llm response details",
		"usage_json", resp.Usage.RawJSON(),
		"raw_response", resp.OutputText())

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

func responseInput(system, user string) responses.ResponseNewParamsInputUnion {
	return responses.ResponseNewParamsInputUnion{
		OfInputItemList: responses.ResponseInputParam{
			systemMessage(system),
			userMessage(user),
		},
	}
}

func webSearchTool() []responses.ToolUnionParam {
	return []responses.ToolUnionParam{
		responses.ToolParamOfWebSearch("web_search"),
		responses.ToolParamOfWebSearch("x_search"),
	}
}
