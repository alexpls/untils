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
	URLs []string `json:"urls"`
}

func browserTools() []responses.ToolUnionParam {
	return []responses.ToolUnionParam{{
		OfFunction: &responses.FunctionToolParam{
			Name:        browserNavigateToolName,
			Description: openai.String("Use a web browser to navigate to the given URLs and retrieve the pages' contents"),
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

		if len(params.URLs) == 0 {
			return "error: no URLs provided", fmt.Errorf("no URLs provided")
		}

		stats.sitesVisited = append(stats.sitesVisited, params.URLs...)

		var sb strings.Builder
		for _, url := range params.URLs {
			sb.WriteString("# " + url + "\n\n")
			res, err := browser.Navigate(ctx, url)
			if err != nil {
				sb.WriteString("error navigating to page: " + err.Error() + "\n\n")
				continue
			}

			writeBrowserNavigateResult(&sb, res)
			sb.WriteString("\n\n")
		}

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
