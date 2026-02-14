package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/alexpls/untils/internal/browser"
	"github.com/alexpls/untils/internal/logging"
	"github.com/alexpls/untils/internal/models"
	"github.com/alexpls/untils/internal/search"
)

var toolRegistry = map[string]toolBuilder{
	browserNavigateTool.name: browserNavigateTool.build,
	browserClickTool.name:    browserClickTool.build,
	browserWaitTool.name:     browserWaitTool.build,
	searchTool.name:          searchTool.build,
}

// toolContext provides dependencies that tools need for execution.
// It must be created per tool call.
type toolContext struct {
	ctx        context.Context
	service    *Service
	browser    func() *browser.BrowserCtx
	priorCalls *[]toolCall
}

// toolCall holds a tool call ready for execution with pre-parsed params.
type toolCall struct {
	call     func() (string, error)
	validate func() string
	params   any
}

// toolBuilder builds a prepared tool call from raw JSON args.
type toolBuilder func(tc *toolContext, args string) (*toolCall, error)

type tool[P any] struct {
	name        string
	description string
	execute     func(tc *toolContext, params P) (string, error)
	validate    func(tc *toolContext, params P) string
}

func (t tool[P]) definition() ToolDefinition {
	var zero P
	return ToolDefinition{
		Name:        t.name,
		Description: t.description,
		Parameters:  jsonSchema(zero),
	}
}

// build parses the JSON args once and returns a prepared call.
func (t tool[P]) build(tc *toolContext, args string) (*toolCall, error) {
	var params P
	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return nil, fmt.Errorf("unmarshaling %s params: %w", t.name, err)
	}
	return &toolCall{
		call:     func() (string, error) { return t.execute(tc, params) },
		validate: func() string { return t.validate(tc, params) },
		params:   params,
	}, nil
}

// tool definitions

var browserNavigateTool = tool[models.BrowserNavigateParams]{
	name:        "browser_navigate",
	description: "Use a web browser to navigate to the given URL and retrieve the page contents",
	execute: func(tc *toolContext, p models.BrowserNavigateParams) (string, error) {
		tc.service.logger.DebugContext(tc.ctx, "browser_navigate started", "url", p.URL)
		start := time.Now()

		llmEvent, _ := logging.GetOrCreateFromContext(tc.ctx, newLLMEvent)
		llmEvent.addSiteVisited(p.URL)

		getBrowserStart := time.Now()
		b := tc.browser()
		tc.service.logger.DebugContext(tc.ctx, "browser_navigate got browser context", "duration", time.Since(getBrowserStart))

		navigateStart := time.Now()
		res, err := b.Navigate(p.URL)
		tc.service.logger.DebugContext(tc.ctx, "browser_navigate navigation completed", "duration", time.Since(navigateStart))

		if err != nil {
			return "", err
		}

		tc.service.logger.DebugContext(tc.ctx, "browser_navigate completed", "url", p.URL, "total_duration", time.Since(start))
		return res.String(), nil
	},
	validate: func(tc *toolContext, params models.BrowserNavigateParams) string {
		return noDuplicateCallsValidator(tc, params, "navigating to the same url multiple times is not allowed. try browsing to a different url")
	},
}

type browserClickParams struct {
	NodeID string `json:"node_id"`
}

var browserClickTool = tool[browserClickParams]{
	name:        "browser_click",
	description: "Use a web browser to click on an element on the current page, identified by its unique ID (e.g. [learn more](click:123) - the ID is 123)",
	execute: func(tc *toolContext, p browserClickParams) (string, error) {
		tc.service.logger.DebugContext(tc.ctx, "browser_click started", "node_id", p.NodeID)
		start := time.Now()

		b := tc.browser()
		clickStart := time.Now()
		page, err := b.Click(p.NodeID)
		tc.service.logger.DebugContext(tc.ctx, "browser_click click completed", "duration", time.Since(clickStart))

		if err != nil {
			tc.service.logger.ErrorContext(tc.ctx, "error performing click", "node_id", p.NodeID, "error", err)
			return "", err
		}

		tc.service.logger.DebugContext(tc.ctx, "browser_click completed", "node_id", p.NodeID, "total_duration", time.Since(start))
		return page.String(), nil
	},
	validate: func(tc *toolContext, params browserClickParams) string {
		return ""
	},
}

type browserWaitParams struct{}

func (p browserWaitParams) equalTo(other any) bool {
	_, ok := other.(browserWaitParams)
	return ok
}

var browserWaitTool = tool[browserWaitParams]{
	name:        "browser_wait",
	description: "Wait for the current page to finish loading. Use this when you suspect dynamic content may not have loaded yet. Returns the updated page contents after waiting.",
	execute: func(tc *toolContext, p browserWaitParams) (string, error) {
		tc.service.logger.DebugContext(tc.ctx, "browser_wait started")
		start := time.Now()

		time.Sleep(3 * time.Second)

		b := tc.browser()
		page, err := b.CurrentPage()
		if err != nil {
			tc.service.logger.ErrorContext(tc.ctx, "error getting current page after wait", "error", err)
			return "", err
		}

		tc.service.logger.DebugContext(tc.ctx, "browser_wait completed", "total_duration", time.Since(start))
		return page.String(), nil
	},
	validate: func(tc *toolContext, params browserWaitParams) string {
		l := len(*tc.priorCalls)
		if l > 0 {
			last := (*tc.priorCalls)[l-1]
			if params.equalTo(last.params) {
				return "can't wait multiple times consecutively. try using another tool " +
					"to navigate to a new page."
			}
		}
		return ""
	},
}

var searchTool = tool[models.SearchRequestParams]{
	name:        "search_request",
	description: "Use a web search engine to search for the given query and retrieve relevant results",
	execute: func(tc *toolContext, p models.SearchRequestParams) (string, error) {
		tc.service.logger.DebugContext(tc.ctx, "search_request started", "query", p.Query)
		start := time.Now()

		res, err := tc.service.webSearcher.Search(search.NewSearchParams(p.Query))
		tc.service.logger.DebugContext(tc.ctx, "search_request search completed", "duration", time.Since(start))

		if err != nil {
			return "", fmt.Errorf("performing search: %w", err)
		}

		var sb strings.Builder
		sb.WriteString("## Search results for query: " + p.Query + "\n\n")
		for _, result := range res.Results {
			sb.WriteString("- " + result.String() + "\n")
		}

		tc.service.logger.DebugContext(tc.ctx, "search_request completed", "query", p.Query, "total_duration", time.Since(start))
		return sb.String(), nil
	},
	validate: func(tc *toolContext, params models.SearchRequestParams) string {
		return noDuplicateCallsValidator(tc, params, "searching with the same query twice is not allowed. "+
			"try adjusting the query or using an existing result from a previous search")
	},
}

// noDuplicateCallsValidator returns a validation error if params matches any prior call.
func noDuplicateCallsValidator[P models.ToolParams](tc *toolContext, params P, errMsg string) string {
	for _, prior := range *tc.priorCalls {
		if params.Equal(prior.params) {
			return errMsg
		}
	}
	return ""
}
