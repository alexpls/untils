package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

const requestRetryDelay = 500 * time.Millisecond

type conversation interface {
	currentTurn() int
	nextTurn() int
	messagesForRequest() []Message
	addSystem(ctx context.Context, content string) error
	addAssistant(ctx context.Context, resp *CompletionResponse, duration time.Duration) error
	addToolOutput(ctx context.Context, call ToolCall, output string) error
}

type toolExecutor func(ctx context.Context, call ToolCall) (string, error)

type agentRunOptions[T any] struct {
	model             string
	responseName      string
	responseSchema    map[string]any
	tools             []ToolDefinition
	parallelToolCalls bool
	maxTurns          int
	conversation      conversation
	toolExecutor      toolExecutor
	validate          func(*T) string
}

func runAgent[T any](ctx context.Context, service *Service, opts agentRunOptions[T]) (*T, error) {
	var lastErr error

	for {
		if opts.conversation.currentTurn() >= opts.maxTurns {
			if lastErr != nil {
				return nil, fmt.Errorf("exceeded max turns (%d): %w", opts.maxTurns, lastErr)
			}
			return nil, fmt.Errorf("exceeded max turns (%d)", opts.maxTurns)
		}

		turn := opts.conversation.nextTurn()
		turnStart := time.Now()

		service.logger.DebugContext(ctx, "starting turn", "turn", turn)

		llmStart := time.Now()
		resp, err := service.response(ctx, CompletionRequest{
			Model:             opts.model,
			Messages:          opts.conversation.messagesForRequest(),
			ResponseName:      opts.responseName,
			ResponseSchema:    opts.responseSchema,
			Tools:             opts.tools,
			ParallelToolCalls: opts.parallelToolCalls,
		})
		llmDuration := time.Since(llmStart)

		if err != nil {
			lastErr = err
			service.logger.ErrorContext(ctx, "error getting response", "turn", turn, "duration", llmDuration, "error", err)
			if turn >= opts.maxTurns {
				return nil, err
			}
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(requestRetryDelay):
			}
			continue
		}

		service.logger.DebugContext(ctx, "response received", "turn", turn, "duration", llmDuration)

		if persistErr := opts.conversation.addAssistant(ctx, resp, llmDuration); persistErr != nil {
			return nil, fmt.Errorf("persisting assistant message: %w", persistErr)
		}

		if len(resp.ToolCalls) > 0 {
			if opts.toolExecutor == nil {
				return nil, fmt.Errorf("provider returned tool calls but no tool executor configured")
			}

			toolsStart := time.Now()
			for _, call := range resp.ToolCalls {
				result, toolErr := opts.toolExecutor(ctx, call)
				if toolErr != nil {
					service.logger.ErrorContext(ctx, "error executing tool call", "tool", call.Name, "error", toolErr)
					result = "error: " + toolErr.Error()
				}

				if persistErr := opts.conversation.addToolOutput(ctx, call, result); persistErr != nil {
					return nil, fmt.Errorf("persisting tool output message: %w", persistErr)
				}
			}

			service.logger.DebugContext(ctx, "turn completed with tool calls",
				"turn", turn,
				"tools_duration", time.Since(toolsStart),
				"turn_duration", time.Since(turnStart),
			)
			continue
		}

		output := sanitizeXAIOutput(resp.Output)

		var out T
		if err := json.Unmarshal([]byte(output), &out); err != nil {
			lastErr = err
			errMsg := fmt.Sprintf(
				"error: output was not valid JSON: %s. ensure your response follows the correct JSON schema.",
				err.Error(),
			)
			if persistErr := opts.conversation.addSystem(ctx, errMsg); persistErr != nil {
				return nil, fmt.Errorf("persisting system message: %w", persistErr)
			}
			continue
		}

		if opts.validate != nil {
			if errMsg := opts.validate(&out); errMsg != "" {
				if persistErr := opts.conversation.addSystem(ctx, errMsg); persistErr != nil {
					return nil, fmt.Errorf("persisting system message: %w", persistErr)
				}
				continue
			}
		}

		return &out, nil
	}
}
