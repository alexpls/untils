package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

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
			Description: openai.String("Use a web browser to navigate to the given URL and retrieve the page contents"),
			Parameters:  jsonSchema(browserNavigateToolParams{}),
		},
	}}
}

func handleToolCall(ctx context.Context, name string, args string) (string, error) {
	stats := statsFromContext(ctx)

	switch name {
	case browserNavigateToolName:
		var params browserNavigateToolParams
		if err := json.Unmarshal([]byte(args), &params); err != nil {
			return "error parsing arguments", fmt.Errorf("parsing arguments: %w", err)
		}

		stats.sitesVisited = append(stats.sitesVisited, params.URL)

		var sb strings.Builder
		sb.WriteString("# " + params.URL + "\n\n")
		res, err := browser.Navigate(ctx, params.URL)
		if err != nil {
			sb.WriteString("error navigating to page: " + err.Error() + "\n\n")
			return sb.String(), nil
		}

		writeBrowserNavigateResult(&sb, res)

		return sb.String(), nil
	default:
		return "tool does not exist", fmt.Errorf("tool does not exist: %s", name)
	}
}

func writeBrowserNavigateResult(sb *strings.Builder, res *browser.NavigateResult) {
	sb.WriteString(`## Page title
		` + res.Page.Title + `

		## Page body
		` + res.Page.Contents)
}
