package llm

import (
	"context"
	_ "embed"
	"fmt"
	"strings"
	"time"

	"github.com/alexpls/untils/internal/browser"
	"github.com/alexpls/untils/internal/models"
)

type checker struct {
	service       *Service
	conversation  *dbConversation
	browserCtx    *browser.BrowserCtx
	browserCancel context.CancelFunc
	priorCalls    []toolCall
}

func newChecker(service *Service) *checker {
	return &checker{
		service:      service,
		conversation: newDBConversation(service),
	}
}

//go:embed checker_prompt.md
var checkerPrompt string

func (c *checker) perform(ctx context.Context, params *CheckParams) (*models.CheckResult, error) {
	var err error

	workflowStart := time.Now()
	c.service.logger.DebugContext(ctx, "checker workflow started")

	var previousResult *models.GetPreviousResultsWithCheckRow
	if len(params.PreviousResults) > 0 {
		previousResult = params.PreviousResults[0]
	}

	if err = c.conversation.start(ctx, params.UserID, params.MonitorCheckID, models.LlmConversationsSourceCheck); err != nil {
		return nil, err
	}

	defer func() {
		if err == nil {
			c.service.logger.DebugContext(ctx, "checker workflow completed", "total_duration", time.Since(workflowStart))
		} else {
			c.service.logger.ErrorContext(ctx, "error during checker workflow", "total_duration", time.Since(workflowStart), "error", err)
		}
		if c.browserCancel != nil {
			c.browserCancel()
		}
	}()

	systemMsg := checkerPrompt
	userMsg := params.UserMessageString()

	if logErr := c.conversation.addSystem(ctx, systemMsg); logErr != nil {
		return nil, fmt.Errorf("failed to log system message: %w", logErr)
	}
	if logErr := c.conversation.addUser(ctx, userMsg); logErr != nil {
		return nil, fmt.Errorf("failed to log user message: %w", logErr)
	}

	res, runErr := runAgent[models.CheckResult](ctx, c.service, agentRunOptions[models.CheckResult]{
		model:          modelReasoning,
		responseName:   "CheckResult",
		responseSchema: jsonSchema(models.CheckResult{}),
		tools: []ToolDefinition{
			browserWaitTool.definition(),
			browserNavigateTool.definition(),
			browserClickTool.definition(),
			searchTool.definition(),
		},
		parallelToolCalls: false,
		maxTurns:          99,
		conversation:      c.conversation,
		toolExecutor:      c.executeToolCall,
		validate: func(res *models.CheckResult) string {
			if previousResult == nil {
				return ""
			}
			if res.DifferentToPrevious && sameResultStr(res.ResultPlaintext, previousResult.MonitorResultsWithLatestCheck.Result) {
				return "error: different_to_previous is true but result_plaintext is the same as the previous result"
			}
			return ""
		},
	})
	err = runErr
	return res, runErr
}

func (c *checker) executeToolCall(ctx context.Context, call ToolCall) (string, error) {
	name := call.Name
	args := call.Arguments

	toolStart := time.Now()
	c.service.logger.DebugContext(ctx, "tool call started", "tool", name, "args", args)

	tc := &toolContext{
		ctx:        ctx,
		service:    c.service,
		priorCalls: &c.priorCalls,
		browser: func() *browser.BrowserCtx {
			if c.browserCtx == nil {
				browserInitStart := time.Now()
				bctx, bcancel := browser.NewBrowser(ctx, c.service.logger)
				c.browserCtx, c.browserCancel = &bctx, bcancel
				c.service.logger.DebugContext(ctx, "browser context initialized", "duration", time.Since(browserInitStart))
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

	if validation := tool.validate(); validation != "" {
		return "", fmt.Errorf("error: %s", validation)
	}

	result, err := tool.call()
	if err == nil {
		c.priorCalls = append(c.priorCalls, *tool)
	}
	c.service.logger.DebugContext(ctx, "tool call completed", "tool", name, "duration", time.Since(toolStart))
	return result, err
}

func sameResultStr(a, b string) bool {
	san := func(s string) string {
		return strings.ToLower(strings.Trim(s, " "))
	}
	return san(a) == san(b)
}
