package llm

import (
	"context"
	_ "embed"
	"fmt"
	"strings"
	"time"

	"github.com/alexpls/untils/internal/browser"
	"github.com/alexpls/untils/internal/llm/instructions"
	llmtools "github.com/alexpls/untils/internal/llm/tools"
	"github.com/alexpls/untils/internal/logging"
	"github.com/alexpls/untils/internal/models"
)

type checker struct {
	service        *Service
	conversation   *dbConversation
	browserSession *browser.BrowserSession
	browserCancel  context.CancelFunc
	priorCalls     []llmtools.Call
}

func newChecker(service *Service) *checker {
	return &checker{
		service:      service,
		conversation: newDBConversation(service),
	}
}

//go:embed checker_prompt.md
var checkerPrompt string

func (c *checker) perform(ctx context.Context, params *CheckParams) (*models.CheckResultWithSchema, error) {
	var err error

	workflowStart := time.Now()
	c.service.logger.DebugContext(ctx, "checker workflow started")

	previousResult := models.LatestVisiblePreviousResult(params.PreviousResults)
	isFirstCheck := len(params.PreviousResults) == 0

	var responseSchema map[string]any
	if params.Schema.Zero() {
		responseSchema = jsonSchema(models.CheckResultWithSchema{})
	} else {
		responseSchema = jsonSchema(models.CheckResult{})
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

	toolUsage := llmtools.Registry.UsageMarkdown()
	systemMsg := strings.ReplaceAll(checkerPrompt, "{{TOOLS_USAGE}}", toolUsage) +
		"\n## Instructions index\n\n" + instructions.Registry.Index()
	userMsg := params.UserMessageString()

	if logErr := c.conversation.addSystem(ctx, systemMsg); logErr != nil {
		return nil, fmt.Errorf("failed to log system message: %w", logErr)
	}
	if logErr := c.conversation.addUser(ctx, userMsg); logErr != nil {
		return nil, fmt.Errorf("failed to log user message: %w", logErr)
	}

	return runAgent(ctx, c.service, agentRunOptions[models.CheckResultWithSchema]{
		model:             c.service.model,
		responseName:      "CheckResult",
		responseSchema:    responseSchema,
		tools:             toProviderTools(llmtools.Registry.Definitions()),
		parallelToolCalls: false,
		maxTurns:          99,
		conversation:      c.conversation,
		toolExecutor:      c.executeToolCall,
		validate: func(res *models.CheckResultWithSchema) string {
			if params.Schema.Zero() {
				if res.Success && res.Schema.Zero() {
					return "error: schema: must be provided"
				}

				if err := res.Schema.Validate(); err != nil {
					return "error: schema: " + err.Error()
				}
			}

			if res.Success && len(res.Updates) == 0 {
				return "error: updates: must contain at least one item when success is true"
			}

			if err := res.Updates.Validate(); err != nil {
				return "error: updates: " + err.Error()
			}

			var schemaForValidation models.MonitorSchemaData
			if !res.Schema.Zero() {
				schemaForValidation = res.Schema
			} else if !params.Schema.Zero() {
				schemaForValidation = params.Schema
			} else {
				panic("no schema either in the response or in the params of this check")
			}

			if err := res.Updates.ValidateAgainstSchema(schemaForValidation); err != nil {
				return "error: updates: " + err.Error()
			}

			if firstCheckUpdateCountMismatch(res, isFirstCheck) {
				return "error: first check must return exactly one update when success is true"
			}

			if duplicateUpdatesMismatch(res) {
				return "error: updates: duplicate updates are not allowed"
			}

			if differentToPreviousMismatch(res, previousResult) {
				return "error: different_to_previous is true but returned update fields are the same as the previous result"
			}

			return ""
		},
	})
}

func firstCheckUpdateCountMismatch(res *models.CheckResultWithSchema, isFirstCheck bool) bool {
	if !isFirstCheck || !res.Success {
		return false
	}

	return len(res.Updates) != 1
}

func duplicateUpdatesMismatch(res *models.CheckResultWithSchema) bool {
	if len(res.Updates) < 2 {
		return false
	}

	for i := 0; i < len(res.Updates); i++ {
		for j := i + 1; j < len(res.Updates); j++ {
			if models.MonitorUpdateFieldsEqual(res.Updates[i].Fields, res.Updates[j].Fields) {
				return true
			}
		}
	}

	return false
}

func differentToPreviousMismatch(res *models.CheckResultWithSchema, previousResult *models.GetPreviousResultsWithCheckRow) bool {
	if previousResult == nil || !res.DifferentToPrevious || len(res.Updates) == 0 {
		return false
	}

	for _, update := range res.Updates {
		if !models.MonitorUpdateFieldsEqual(update.Fields, previousResult.MonitorResult.Data.Fields) {
			return false
		}
	}

	return true
}

func (c *checker) executeToolCall(ctx context.Context, call ToolCall) (string, error) {
	name := call.Name
	args := call.Arguments

	toolStart := time.Now()
	c.service.logger.DebugContext(ctx, "tool call started", "tool", name, "args", args)

	llmEvent, _ := logging.GetOrCreateFromContext(ctx, newLLMEvent)

	tc := &llmtools.Context{
		Ctx:        ctx,
		Logger:     c.service.logger,
		PriorCalls: &c.priorCalls,
		Search:     c.service.webSearcher.Search,
		ReadInstruction: func(name string) (string, error) {
			return instructions.Registry.Body(name)
		},
		AddSiteVisited: llmEvent.addSiteVisited,
		Browser: func() (*browser.BrowserSession, error) {
			if c.browserSession == nil {
				browserInitStart := time.Now()
				session, sessionCancel, err := c.service.newBrowserSession(ctx)
				if err != nil {
					return nil, err
				}
				c.browserSession, c.browserCancel = &session, sessionCancel
				c.service.logger.DebugContext(ctx, "browser context initialized", "duration", time.Since(browserInitStart))
			}
			return c.browserSession, nil
		},
	}

	tool, err := llmtools.Registry.Build(name, tc, args)
	if err != nil {
		return "", err
	}

	if validation := tool.Validate(); validation != "" {
		return "", fmt.Errorf("error: %s", validation)
	}

	result, err := tool.Execute()
	if err == nil {
		c.priorCalls = append(c.priorCalls, *tool)
	}
	c.service.logger.DebugContext(ctx, "tool call completed", "tool", name, "duration", time.Since(toolStart))
	return result, err
}
