package llm

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/openai/openai-go/v3/responses"
)

//go:embed source_finder_prompt.md
var sourceFinderPrompt string

type sourceFinder struct {
	service  *Service
	messages []responses.ResponseInputItemUnionParam
}

func newSourceFinder(service *Service) *sourceFinder {
	return &sourceFinder{service: service}
}

type Source struct {
	URL         string `json:"url"`
	Monitorable bool   `json:"monitorable"`
}

type sourceFinderResponse struct {
	Success       bool     `json:"success"`
	FailureReason string   `json:"failure_reason"`
	Sources       []Source `json:"sources"`
}

func (p *sourceFinder) Run(ctx context.Context, params *CheckParams) (*sourceFinderResponse, error) {
	p.messages = append(p.messages,
		systemMessage(sourceFinderPrompt),
		userMessage("Subject: "+params.Subject+
			"\n\nInstructions: "+params.Instructions),
	)

	var turn int
	maxTurns := 3
	var resp *responseResult
	var err error

	for {
		turn++
		if turn > maxTurns {
			return nil, fmt.Errorf("max tries exceeded: %w", err)
		}

		resp, err = p.service.response(ctx, responses.ResponseNewParams{
			Model: model,
			Input: inputItems(p.messages...),
			Text:  jsonSchemaResponse(sourceFinderResponse{}),
			Tools: webSearchTools(),
		})
		if err != nil {
			return nil, err
		}

		var result sourceFinderResponse
		err = json.Unmarshal([]byte(resp.OutputText()), &result)
		if err != nil {
			p.messages = append(p.messages, systemMessage("Error: json was invalid"))
			continue
		}

		if len(result.Sources) == 0 {
			p.messages = append(p.messages, systemMessage("Error: no sources found. "+
				"If you can't find any, respond with success: false and provide a failure_reason"))
			continue
		}

		if len(result.Sources) > 0 && !result.Success {
			p.messages = append(p.messages, systemMessage("Error: sources were found but success is false"))
			continue
		}

		return &result, nil
	}
}
