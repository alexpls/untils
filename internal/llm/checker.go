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
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
)

type checker struct {
	service        *Service
	messages       []responses.ResponseInputItemUnionParam
	browserCtx     *browser.BrowserCtx
	browserCancel  context.CancelFunc
	c              EventsChan
	pool           *pgxpool.Pool
	queries        *models.Queries
	conversationID int64
	turn           int
}

func newChecker(service *Service, c EventsChan, pool *pgxpool.Pool, queries *models.Queries) *checker {
	return &checker{service: service, c: c, pool: pool, queries: queries}
}

func (c *checker) logMessage(ctx context.Context, role models.LLMMessageRole, body any, duration time.Duration) error {
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshalling message body: %w", err)
	}
	return c.queries.AddMessageToLLMConversation(ctx, c.pool, &models.AddMessageToLLMConversationParams{
		LlmConversationID: c.conversationID,
		Message: models.LLMConversationMessages{
			{
				Turn:     c.turn,
				Role:     role,
				At:       time.Now(),
				Duration: duration,
				Body:     bodyJSON,
			},
		},
	})
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

	conversation, err := c.queries.CreateLLMConversation(ctx, c.pool, &models.CreateLLMConversationParams{
		UserID:     params.UserID,
		SourceType: models.LlmConversationsSourceCheck,
		SourceID:   params.MonitorCheckID,
	})
	if err != nil {
		return nil, fmt.Errorf("creating llm conversation: %w", err)
	}
	c.conversationID = conversation.ID

	defer func() {
		c.service.logger.Debug("checker workflow completed", "total_duration", time.Since(workflowStart))
		if c.browserCancel != nil {
			c.browserCancel()
		}
	}()

	systemMsg := checkerPrompt
	userMsg := params.UserMessageString()

	c.messages = []responses.ResponseInputItemUnionParam{
		systemMessage(systemMsg),
		userMessage(userMsg),
	}

	if logErr := c.logMessage(ctx, models.LLMMessageRoleSystem, map[string]string{"content": systemMsg}, 0); logErr != nil {
		return nil, fmt.Errorf("failed to log system message: %w", logErr)
	}
	if logErr := c.logMessage(ctx, models.LLMMessageRoleUser, map[string]string{"content": userMsg}, 0); logErr != nil {
		return nil, fmt.Errorf("failed to log user message: %w", logErr)
	}

	var resp *responseResult
	maxTurns := 99

	for {
		if c.turn >= maxTurns {
			return nil, fmt.Errorf("exceeded max turns: %w", err)
		}
		c.turn++
		turnStart := time.Now()
		c.service.logger.Debug("starting turn", "turn", c.turn)

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
		llmDuration := time.Since(llmStart)
		c.service.logger.Debug("LLM response received", "turn", c.turn, "duration", llmDuration)

		if resp != nil {
			if logErr := c.logMessage(ctx, models.LLMMessageRoleAssistant, json.RawMessage(resp.RawJSON()), llmDuration); logErr != nil {
				return nil, fmt.Errorf("failed to log assistant message: %w", logErr)
			}
		}

		if err != nil {
			return nil, err
		}

		if len(resp.toolCalls) > 0 {
			toolsStart := time.Now()
			if err := c.callTools(ctx, resp.toolCalls); err != nil {
				return nil, err
			}
			c.service.logger.Debug("turn completed with tool calls", "turn", c.turn, "tools_duration", time.Since(toolsStart), "turn_duration", time.Since(turnStart))
			continue
		}

		sanitized := sanitizeXAIOutput(resp.OutputText())
		res := &models.CheckResult{}
		if err := json.Unmarshal([]byte(sanitized), res); err != nil {
			errorMsg := "error: invalid response format"
			c.messages = append(c.messages, systemMessage(errorMsg))
			if logErr := c.logMessage(ctx, models.LLMMessageRoleSystem, map[string]string{"content": errorMsg}, 0); logErr != nil {
				return nil, fmt.Errorf("failed to log error message: %w", logErr)
			}
			continue
		}

		if previousResult != nil {
			if res.DifferentToPrevious && sameResultStr(res.ResultPlaintext, previousResult.MonitorResult.Result) {
				errorMsg := "error: different_to_previous is true but result_plaintext is the same as the previous result"
				c.messages = append(
					c.messages,
					systemMessage(errorMsg),
				)
				if logErr := c.logMessage(ctx, models.LLMMessageRoleSystem, map[string]string{"content": errorMsg}, 0); logErr != nil {
					return nil, fmt.Errorf("failed to log error message: %w", logErr)
				}
				continue
			}
		}

		return res, nil
	}
}

func (c *checker) callTools(ctx context.Context, toolCalls []responses.ResponseFunctionToolCall) error {
	for _, call := range toolCalls {
		result, err := c.callTool(ctx, call.Name, call.Arguments)
		if err != nil {
			c.service.logger.Error("error executing tool call", "tool", call.Name, "error", err)
			errorMsg := "error executing tool call: " + err.Error()
			c.messages = append(c.messages, systemMessage(errorMsg))
			if logErr := c.logMessage(ctx, models.LLMMessageRoleSystem, map[string]string{"content": errorMsg}, 0); logErr != nil {
				return fmt.Errorf("failed to log tool error message: %w", logErr)
			}
			continue
		}
		c.messages = append(c.messages, toolOutputMessage(call.ID, result))
		if logErr := c.logMessage(ctx, models.LLMMessageRoleTool, map[string]any{
			"call_id": call.ID,
			"name":    call.Name,
			"output":  result,
		}, 0); logErr != nil {
			return fmt.Errorf("failed to log tool output message: %w", logErr)
		}
	}
	return nil
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
