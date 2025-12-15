package llm

import (
	"log/slog"

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

func responseInput(system, user string) responses.ResponseNewParamsInputUnion {
	return responses.ResponseNewParamsInputUnion{
		OfInputItemList: responses.ResponseInputParam{
			responses.ResponseInputItemUnionParam{
				OfMessage: &responses.EasyInputMessageParam{
					Content: responses.EasyInputMessageContentUnionParam{
						OfString: openai.String(system),
					},
					Role: "system",
				},
			},
			responses.ResponseInputItemUnionParam{
				OfMessage: &responses.EasyInputMessageParam{
					Content: responses.EasyInputMessageContentUnionParam{
						OfString: openai.String(user),
					},
					Role: "user",
				},
			},
		},
	}
}

func webSearchTool() []responses.ToolUnionParam {
	return []responses.ToolUnionParam{
		responses.ToolParamOfWebSearch("web_search"),
		responses.ToolParamOfWebSearch("x_search"),
	}
}
