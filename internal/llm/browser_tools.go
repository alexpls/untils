package llm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/alexpls/untils_go/internal/browser"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
)

const (
	browserNavigateToolName = "browser_navigate"
)

type browserNavigateToolParams struct {
	URL string `json:"url"`
}

func browserTools() []responses.ToolUnionParam {
	return []responses.ToolUnionParam{{
		OfFunction: &responses.FunctionToolParam{
			Name:        browserNavigateToolName,
			Description: openai.String("Use a web browser to navigate to the given URL and retrieve the page's contents"),
			Parameters:  jsonSchema(browserNavigateToolParams{}),
		},
	}}
}

func handleToolCall(ctx context.Context, name string, args string) (string, error) {
	switch name {
	case browserNavigateToolName:
		var params browserNavigateToolParams
		if err := json.Unmarshal([]byte(args), &params); err != nil {
			return "error parsing arguments", fmt.Errorf("parsing arguments: %w", err)
		}

		res, err := browser.Navigate(ctx, params.URL)
		if err != nil {
			return "error navigating to page", fmt.Errorf("browser navigating to page: %w", err)
		}

		return `## Navigation results

		### Page title
		` + res.Page.Title + `

		### Page body (accessibility tree)
		` + res.Page.Contents + `
		`, nil
	default:
		return "tool does not exist", fmt.Errorf("tool does not exist: %s", name)
	}
}
