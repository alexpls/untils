package llm

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/alexpls/untils/internal/browser"
	"github.com/alexpls/untils/internal/models"
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

func (c *checker) perform(ctx context.Context, params *CheckParams) (*models.CheckResult, error) {
	workflowStart := time.Now()
	c.service.logger.Debug("checker workflow started")

	var previousResult *models.GetPreviousResultsWithCheckRow
	if len(params.PreviousResults) > 0 {
		previousResult = params.PreviousResults[0]
	}

	defer func() {
		c.service.logger.Debug("checker workflow completed", "total_duration", time.Since(workflowStart))
		if c.browserCancel != nil {
			c.browserCancel()
		}
	}()
	var err error

	c.messages = []responses.ResponseInputItemUnionParam{
		systemMessage(checkerPrompt),
		userMessage(params.UserMessageString()),
	}

	var resp *responseResult
	maxTurns := 30
	turn := 0

	for {
		if turn >= maxTurns {
			return nil, fmt.Errorf("exceeded max turns: %w", err)
		}
		turn++
		turnStart := time.Now()
		c.service.logger.Debug("starting turn", "turn", turn)

		llmStart := time.Now()
		resp, err = c.service.response(ctx, responses.ResponseNewParams{
			Model: modelReasoning,
			Input: inputItems(c.messages...),
			Text:  jsonSchemaResponse(models.CheckResult{}),
			Tools: []responses.ToolUnionParam{
				browserNavigateTool.toOpenAIParam(),
				browserClickTool.toOpenAIParam(),
				searchTool.toOpenAIParam(),
			},
			ParallelToolCalls: openai.Bool(false),
		})
		c.service.logger.Debug("LLM response received", "turn", turn, "duration", time.Since(llmStart))

		if len(resp.toolCalls) > 0 {
			toolsStart := time.Now()
			c.callTools(ctx, resp.toolCalls)
			c.service.logger.Debug("turn completed with tool calls", "turn", turn, "tools_duration", time.Since(toolsStart), "turn_duration", time.Since(turnStart))
			continue
		}

		sanitized := sanitizeXAIOutput(resp.OutputText())
		res := &models.CheckResult{}
		if err := json.Unmarshal([]byte(sanitized), res); err != nil {
			c.messages = append(c.messages, systemMessage("error: invalid response format"))
			continue
		}

		if previousResult != nil {
			if res.DifferentToPrevious && sameResultStr(res.ResultPlaintext, previousResult.MonitorResult.Result) {
				c.messages = append(
					c.messages,
					systemMessage("error: different_to_previous is true but result_plaintext is the same as the previous result"),
				)
				continue
			}
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
	toolStart := time.Now()
	c.service.logger.Debug("tool call started", "tool", name)

	tc := &toolContext{
		ctx:     ctx,
		service: c.service,
		browser: func() *browser.BrowserCtx {
			if c.browserCtx == nil {
				browserInitStart := time.Now()
				bctx, bcancel := browser.NewBrowser(ctx, c.service.logger)
				c.browserCtx, c.browserCancel = &bctx, bcancel
				c.service.logger.Debug("browser context initialized", "duration", time.Since(browserInitStart))
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

	result, err := tool.call()
	c.service.logger.Debug("tool call completed", "tool", name, "duration", time.Since(toolStart))
	return result, err
}

func sameResultStr(a, b string) bool {
	san := func(s string) string {
		return strings.ToLower(strings.Trim(s, " "))
	}
	return san(a) == san(b)
}
