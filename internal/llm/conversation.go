package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/alexpls/untils/internal/models"
)

type dbConversation struct {
	service *Service

	id       int64
	turn     int
	messages []Message
}

func newDBConversation(service *Service) *dbConversation {
	return &dbConversation{service: service}
}

func (c *dbConversation) start(ctx context.Context, userID, sourceID int64, sourceType models.LLMConversationsSource) error {
	record, err := c.service.queries.CreateLLMConversation(ctx, c.service.db, &models.CreateLLMConversationParams{
		UserID:     userID,
		SourceType: sourceType,
		SourceID:   sourceID,
	})
	if err != nil {
		return fmt.Errorf("creating llm conversation: %w", err)
	}
	c.id = record.ID
	return nil
}

func (c *dbConversation) nextTurn() int {
	c.turn++
	return c.turn
}

func (c *dbConversation) currentTurn() int {
	return c.turn
}

func (c *dbConversation) messagesForRequest() []Message {
	return c.messages
}

func (c *dbConversation) addSystem(ctx context.Context, content string) error {
	c.messages = append(c.messages, Message{
		Role:    MessageRoleSystem,
		Content: content,
	})
	return c.logMessage(ctx, models.LLMMessageRoleSystem, map[string]string{"content": content}, 0)
}

func (c *dbConversation) addUser(ctx context.Context, content string) error {
	c.messages = append(c.messages, Message{
		Role:    MessageRoleUser,
		Content: content,
	})
	return c.logMessage(ctx, models.LLMMessageRoleUser, map[string]string{"content": content}, 0)
}

func (c *dbConversation) addAssistant(ctx context.Context, resp *CompletionResponse, duration time.Duration) error {
	for _, call := range resp.ToolCalls {
		c.messages = append(c.messages, Message{
			Role:     MessageRoleAssistant,
			ToolCall: &call,
		})
	}

	body := map[string]any{
		"text_output": resp.Output,
		"raw":         resp.Raw,
	}

	if len(resp.ToolCalls) > 0 {
		toolCalls := make([]map[string]string, 0, len(resp.ToolCalls))
		for _, call := range resp.ToolCalls {
			toolCalls = append(toolCalls, map[string]string{
				"id":        call.ID,
				"name":      call.Name,
				"arguments": call.Arguments,
			})
		}
		body["tool_calls"] = toolCalls
	}

	return c.logMessage(ctx, models.LLMMessageRoleAssistant, body, duration)
}

func (c *dbConversation) addToolOutput(ctx context.Context, call ToolCall, output string) error {
	c.messages = append(c.messages, Message{
		Role: MessageRoleTool,
		ToolOutput: &ToolOutput{
			CallID: call.ID,
			Output: output,
		},
	})
	return c.logMessage(ctx, models.LLMMessageRoleTool, map[string]any{
		"call_id": call.ID,
		"name":    call.Name,
		"output":  output,
	}, 0)
}

func (c *dbConversation) logMessage(ctx context.Context, role models.LLMMessageRole, body any, duration time.Duration) error {
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshalling message body: %w", err)
	}

	return c.service.queries.AddMessageToLLMConversation(ctx, c.service.db, &models.AddMessageToLLMConversationParams{
		LlmConversationID: c.id,
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
