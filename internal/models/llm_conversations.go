package models

import (
	"encoding/json"
	"time"
)

type LLMConversationMessages []*LLMConversationMessage

type LLMMessageRole string

const (
	LLMMessageRoleSystem    LLMMessageRole = "system"
	LLMMessageRoleUser      LLMMessageRole = "user"
	LLMMessageRoleAssistant LLMMessageRole = "assistant"
	LLMMessageRoleTool      LLMMessageRole = "tool"
)

type LLMConversationMessage struct {
	Turn     int             `json:"turn"`
	Role     LLMMessageRole  `json:"role"`
	At       time.Time       `json:"at"`
	Duration time.Duration   `json:"duration,omitempty"`
	Body     json.RawMessage `json:"body"`
}

// LLMAssistantMessageBody represents the body of an assistant message.
type LLMAssistantMessageBody struct {
	Output []LLMAssistantOutputItem `json:"output"`
}

// LLMAssistantOutputItem represents an item in the assistant message output array.
type LLMAssistantOutputItem struct {
	Type      string `json:"type"`
	CallID    string `json:"call_id,omitempty"`
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

// LLMToolCall represents a tool call extracted from an assistant message.
type LLMToolCall struct {
	Name      string
	Arguments string
	At        time.Time
}

// ToLLMToolCall converts a GetTimelineEventsBySourceIDRow to LLMToolCall.
func (r *GetTimelineEventsBySourceIDRow) ToLLMToolCall() LLMToolCall {
	return LLMToolCall{
		Name:      r.Name,
		Arguments: r.Arguments,
		At:        r.At.Time,
	}
}

// ExtractToolCalls extracts all tool calls from the conversation messages.
// It parses assistant messages and returns tool calls in chronological order.
func (msgs LLMConversationMessages) ExtractToolCalls() []LLMToolCall {
	var calls []LLMToolCall
	for _, msg := range msgs {
		if msg.Role != LLMMessageRoleAssistant {
			continue
		}

		var body LLMAssistantMessageBody
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			continue
		}

		for _, item := range body.Output {
			if item.Type == "function_call" {
				calls = append(calls, LLMToolCall{
					Name:      item.Name,
					Arguments: item.Arguments,
					At:        msg.At,
				})
			}
		}
	}
	return calls
}
