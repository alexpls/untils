package llm

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/alexpls/untils_go/internal/browser"
	"github.com/alexpls/untils_go/internal/search"
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
			Model: "grok-4-1-fast-reasoning",
			Input: inputItems(c.messages...),
			Text:  jsonSchemaResponse(CheckResult{}),
			Tools: append(browserTools(), searchTools()...),
		})

		if len(resp.toolCalls) > 0 {
			c.callTools(ctx, resp.toolCalls)
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

func (c *checker) callTools(ctx context.Context, toolCalls []responses.ResponseFunctionToolCall) {
	for _, call := range toolCalls {
		params, err := toolCallParams(call.Name, call.Arguments)
		if err != nil {
			c.messages = append(c.messages, systemMessage("error: invalid tool call"))
			continue
		}

		result, err := c.callTool(ctx, call.Name, params)
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

func (c *checker) callTool(ctx context.Context, name string, params any) (string, error) {
	stats := statsFromContext(ctx)

	switch p := params.(type) {
	case browserNavigateToolParams:
		// TODO: prevent multiple calls with the same args

		stats.sitesVisited = append(stats.sitesVisited, p.URL)

		var sb strings.Builder
		sb.WriteString("# " + p.URL + "\n\n")
		b, cancel := browser.NewBrowser(ctx)
		defer cancel()
		res, err := b.Navigate(p.URL)
		if err != nil {
			sb.WriteString("error navigating to page: " + err.Error() + "\n\n")
			return sb.String(), nil
		}

		writeBrowserNavigateResult(&sb, res)

		return sb.String(), nil
	case searchToolParams:
		res, err := c.service.webSearcher.Search(search.NewSearchParams(p.Query))
		if err != nil {
			return "", fmt.Errorf("performing search: %w", err)
		}

		var sb strings.Builder
		sb.WriteString("## Search results for query: " + p.Query + "\n\n")
		for _, result := range res.Results {
			sb.WriteString("- " + result.String() + "\n")
		}

		return sb.String(), nil
	default:
		return "tool does not exist", fmt.Errorf("tool does not exist: %s", name)
	}
}

func writeBrowserNavigateResult(sb *strings.Builder, res *browser.Page) {
	sb.WriteString(`## Page title
		` + res.Title + `

		## Page body
		` + res.Contents)
}
