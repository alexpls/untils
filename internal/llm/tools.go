package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/alexpls/untils_go/internal/browser"
	"github.com/alexpls/untils_go/internal/search"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
)

const (
	browserNavigateToolName = "browser_navigate"
	// intentionally different to "web_search" to avoid name collisions with OAI/x.ai
	searchToolName = "search_request"
)

type browserNavigateToolParams struct {
	URL string `json:"url"`
}

type searchToolParams struct {
	Query string `json:"string"`
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

func searchTools() []responses.ToolUnionParam {
	return []responses.ToolUnionParam{{
		OfFunction: &responses.FunctionToolParam{
			Name:        searchToolName,
			Description: openai.String("Use a web search engine to search for the given query and retrieve relevant results"),
			Parameters:  jsonSchema(searchToolParams{}),
		},
	}}
}

func toolCallParams(name, args string) (any, error) {
	switch name {
	case browserNavigateToolName:
		var params browserNavigateToolParams
		if err := json.Unmarshal([]byte(args), &params); err != nil {
			return nil, fmt.Errorf("unmarshaling tool call params: %w", err)
		}
		return params, nil
	case searchToolName:
		var params searchToolParams
		if err := json.Unmarshal([]byte(args), &params); err != nil {
			return nil, fmt.Errorf("unmarshaling tool call params: %w", err)
		}
		return params, nil
	default:
		return nil, fmt.Errorf("tool does not exist: %s", name)
	}
}

func (s *Service) handleToolCall(ctx context.Context, name string, params any) (string, error) {
	stats := statsFromContext(ctx)

	switch p := params.(type) {
	case browserNavigateToolParams:
		// TODO: prevent multiple calls with the same args

		stats.sitesVisited = append(stats.sitesVisited, p.URL)

		var sb strings.Builder
		sb.WriteString("# " + p.URL + "\n\n")
		res, err := browser.Navigate(ctx, p.URL)
		if err != nil {
			sb.WriteString("error navigating to page: " + err.Error() + "\n\n")
			return sb.String(), nil
		}

		writeBrowserNavigateResult(&sb, res)

		return sb.String(), nil
	case searchToolParams:
		res, err := s.webSearcher.Search(search.NewSearchParams(p.Query))
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

func writeBrowserNavigateResult(sb *strings.Builder, res *browser.NavigateResult) {
	sb.WriteString(`## Page title
		` + res.Page.Title + `

		## Page body
		` + res.Page.Contents)
}
