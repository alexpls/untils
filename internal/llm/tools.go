package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/alexpls/untils_go/internal/browser"
	"github.com/alexpls/untils_go/internal/search"
	"github.com/alexpls/untils_go/internal/wideevents"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
)

var toolRegistry = map[string]toolCaller{
	browserNavigateTool.name: browserNavigateTool,
	browserClickTool.name:    browserClickTool,
	searchTool.name:          searchTool,
}

// toolContext provides dependencies that tools need for execution.
// It must be created per request.
type toolContext struct {
	ctx        context.Context
	service    *Service
	getBrowser func() *browser.BrowserCtx
}

type tool[P any] struct {
	name        string
	description string
	execute     func(tc *toolContext, params P) (string, error)
}

// toOpenAIParam returns the tool definition as expected by the OpenAI API.
func (t tool[P]) toOpenAIParam() responses.ToolUnionParam {
	var zero P
	return responses.ToolUnionParam{
		OfFunction: &responses.FunctionToolParam{
			Name:        t.name,
			Description: openai.String(t.description),
			Parameters:  jsonSchema(zero),
		},
	}
}

func (t tool[P]) call(tc *toolContext, args string) (string, error) {
	var params P
	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return "", fmt.Errorf("unmarshaling %s params: %w", t.name, err)
	}
	return t.execute(tc, params)
}

// toolCaller is a non-generic interface for calling tools
type toolCaller interface {
	call(tc *toolContext, args string) (string, error)
}

// tool definitions

type browserNavigateParams struct {
	URL string `json:"url"`
}

var browserNavigateTool = tool[browserNavigateParams]{
	name:        "browser_navigate",
	description: "Use a web browser to navigate to the given URL and retrieve the page contents",
	execute: func(tc *toolContext, p browserNavigateParams) (string, error) {
		llmEvent, _ := wideevents.GetOrCreateFromContext(tc.ctx, newLLMEvent)
		llmEvent.addSiteVisited(p.URL)

		b := tc.getBrowser()
		res, err := b.Navigate(p.URL)
		if err != nil {
			return "", err
		}
		return res.String(), nil
	},
}

type browserClickParams struct {
	NodeID string `json:"node_id"`
}

var browserClickTool = tool[browserClickParams]{
	name:        "browser_click",
	description: "Use a web browser to click on an element on the current page, identified by its unique ID (e.g. [learn more](click:123) - the ID is 123)",
	execute: func(tc *toolContext, p browserClickParams) (string, error) {
		b := tc.getBrowser()
		page, err := b.Click(p.NodeID)
		if err != nil {
			tc.service.logger.Error("error performing click", "node_id", p.NodeID, "error", err)
			return "", err
		}
		return page.String(), nil
	},
}

type searchParams struct {
	Query string `json:"query"`
}

var searchTool = tool[searchParams]{
	name:        "search_request",
	description: "Use a web search engine to search for the given query and retrieve relevant results",
	execute: func(tc *toolContext, p searchParams) (string, error) {
		res, err := tc.service.webSearcher.Search(search.NewSearchParams(p.Query))
		if err != nil {
			return "", fmt.Errorf("performing search: %w", err)
		}

		var sb strings.Builder
		sb.WriteString("## Search results for query: " + p.Query + "\n\n")
		for _, result := range res.Results {
			sb.WriteString("- " + result.String() + "\n")
		}

		return sb.String(), nil
	},
}

func browserTools() []responses.ToolUnionParam {
	return []responses.ToolUnionParam{
		browserNavigateTool.toOpenAIParam(),
		browserClickTool.toOpenAIParam(),
	}
}

func searchTools() []responses.ToolUnionParam {
	return []responses.ToolUnionParam{
		searchTool.toOpenAIParam(),
	}
}

func toolOutputMessage(callID, output string) responses.ResponseInputItemUnionParam {
	return responses.ResponseInputItemUnionParam{
		OfFunctionCallOutput: &responses.ResponseInputItemFunctionCallOutputParam{
			CallID: callID,
			Output: responses.ResponseInputItemFunctionCallOutputOutputUnionParam{
				OfString: openai.String(output),
			},
		},
	}
}
