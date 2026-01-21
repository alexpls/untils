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
