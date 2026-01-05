package llm

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/alexpls/untils/internal/browser"
	"github.com/alexpls/untils/internal/db/sqlc"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
)

type checker struct {
	service       *Service
	messages      []responses.ResponseInputItemUnionParam
	browserCtx    *browser.BrowserCtx
	browserCancel context.CancelFunc
	c             EventsChan
}

func newChecker(service *Service, c EventsChan) *checker {
	return &checker{service: service, c: c}
}

//go:embed checker_prompt.md
var checkerPrompt string

func (c *checker) perform(ctx context.Context, params *CheckParams) (*sqlc.CheckResult, error) {
	defer func() {
		if c.browserCancel != nil {
			c.browserCancel()
		}
	}()
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
	maxTurns := 30
	turn := 0

	for {
		if turn >= maxTurns {
			return nil, fmt.Errorf("exceeded max turns: %w", err)
		}
		turn++

		resp, err = c.service.response(ctx, responses.ResponseNewParams{
			Model: "grok-4-1-fast-reasoning",
			Input: inputItems(c.messages...),
			Text:  jsonSchemaResponse(sqlc.CheckResult{}),
			Tools: []responses.ToolUnionParam{
				browserNavigateTool.toOpenAIParam(),
				browserClickTool.toOpenAIParam(),
				searchTool.toOpenAIParam(),
			},
			ParallelToolCalls: openai.Bool(false),
		})

		if len(resp.toolCalls) > 0 {
			c.callTools(ctx, resp.toolCalls)
			continue
		}

		sanitized := sanitizeXAIOutput(resp.OutputText())
		res := &sqlc.CheckResult{}
		if err := json.Unmarshal([]byte(sanitized), res); err != nil {
			c.messages = append(c.messages, systemMessage("error: invalid response format"))
			continue
		}

		return res, nil
	}
}

func (c *checker) callTools(ctx context.Context, toolCalls []responses.ResponseFunctionToolCall) {
	for _, call := range toolCalls {
		result, err := c.callTool(ctx, call.Name, call.Arguments)
		if err != nil {
			c.service.logger.Error("error executing tool call", "tool", call.Name, "error", err)
			c.messages = append(c.messages, systemMessage("error executing tool call: "+err.Error()))
			continue
		}
		c.messages = append(c.messages, toolOutputMessage(call.ID, result))
	}
}

func (c *checker) callTool(ctx context.Context, name string, args string) (string, error) {
	tc := &toolContext{
		ctx:     ctx,
		service: c.service,
		browser: func() *browser.BrowserCtx {
			if c.browserCtx == nil {
				bctx, bcancel := browser.NewBrowser(ctx, c.service.logger)
				c.browserCtx, c.browserCancel = &bctx, bcancel
			}
			return c.browserCtx
		},
	}

	toolBuilder, ok := toolRegistry[name]
	if !ok {
		return "", fmt.Errorf("tool does not exist: %s", name)
	}

	tool, err := toolBuilder(tc, args)
	if err != nil {
		return "", err
	}

	select {
	case c.c <- tool.checkEvent():
	default:
	}

	return tool.call()
}
