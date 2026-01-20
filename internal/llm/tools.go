package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/alexpls/untils/internal/browser"
	"github.com/alexpls/untils/internal/models"
	"github.com/alexpls/untils/internal/search"
	"github.com/alexpls/untils/internal/wideevents"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
)

var toolRegistry = map[string]toolBuilder{
	browserNavigateTool.name: browserNavigateTool.build,
	browserClickTool.name:    browserClickTool.build,
	searchTool.name:          searchTool.build,
}

// toolContext provides dependencies that tools need for execution.
// It must be created per tool call.
type toolContext struct {
	ctx     context.Context
	service *Service
	browser func() *browser.BrowserCtx
}

// toolCall holds a tool call ready for execution with pre-parsed params.
type toolCall struct {
	call       func() (string, error)
	checkEvent func() CheckEvent
}

// toolBuilder builds a prepared tool call from raw JSON args.
type toolBuilder func(tc *toolContext, args string) (*toolCall, error)

type tool[P any] struct {
	name        string
	description string
	execute     func(tc *toolContext, params P) (string, error)
	checkEvent  func(tc *toolContext, params P) CheckEvent
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

// build parses the JSON args once and returns a prepared call.
func (t tool[P]) build(tc *toolContext, args string) (*toolCall, error) {
	var params P
	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return nil, fmt.Errorf("unmarshaling %s params: %w", t.name, err)
	}
	return &toolCall{
		call:       func() (string, error) { return t.execute(tc, params) },
		checkEvent: func() CheckEvent { return t.checkEvent(tc, params) },
	}, nil
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

		b := tc.browser()
		res, err := b.Navigate(p.URL)
		if err != nil {
			return "", err
		}
		return res.String(), nil
	},
	checkEvent: func(tc *toolContext, params browserNavigateParams) CheckEvent {
		return CheckEvent{
			Kind: models.MonitorCheckEventKindBrowserNavigate,
			Details: models.MonitorCheckEventBrowserNavigateDetails{
				URL: params.URL,
			},
		}
	},
}

type browserClickParams struct {
	NodeID string `json:"node_id"`
}

var browserClickTool = tool[browserClickParams]{
	name:        "browser_click",
	description: "Use a web browser to click on an element on the current page, identified by its unique ID (e.g. [learn more](click:123) - the ID is 123)",
	execute: func(tc *toolContext, p browserClickParams) (string, error) {
		b := tc.browser()
		page, err := b.Click(p.NodeID)
		if err != nil {
			tc.service.logger.Error("error performing click", "node_id", p.NodeID, "error", err)
			return "", err
		}
		return page.String(), nil
	},
	checkEvent: func(tc *toolContext, params browserClickParams) CheckEvent {
		return CheckEvent{
			Kind:    models.MonitorCheckEventKindBrowserClick,
			Details: models.MonitorCheckEventBrowserClickDetails{},
		}
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
	checkEvent: func(tc *toolContext, params searchParams) CheckEvent {
		return CheckEvent{
			Kind: models.MonitorCheckEventKindWebSearch,
			Details: models.MonitorCheckEventWebSearchDetails{
				Query: params.Query,
			},
		}
	},
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
