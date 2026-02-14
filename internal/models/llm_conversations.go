package models

import (
	"bytes"
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

// LLMAssistantNormalizedMessageBody is the provider-neutral assistant message
// shape stored by the llm package.
type LLMAssistantNormalizedMessageBody struct {
	TextOutput string                           `json:"text_output,omitempty"`
	ToolCalls  []LLMAssistantNormalizedToolCall `json:"tool_calls,omitempty"`
}

type LLMAssistantNormalizedToolCall struct {
	ID        string `json:"id,omitempty"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// LLMAssistantContent represents the parsed content from an assistant message,
// providing easy access to tool calls and text output.
type LLMAssistantContent struct {
	ToolCalls  []LLMToolCall
	TextOutput string
}

// parseAssistantMessageBody parses an assistant message body from JSON and
// returns structured content with tool calls and text output extracted.
// Returns nil if the body cannot be parsed.
func parseAssistantMessageBody(body json.RawMessage) *LLMAssistantContent {
	var data LLMAssistantNormalizedMessageBody
	if err := json.Unmarshal(body, &data); err != nil {
		return nil
	}

	result := &LLMAssistantContent{
		TextOutput: data.TextOutput,
		ToolCalls:  make([]LLMToolCall, 0, len(data.ToolCalls)),
	}
	for _, call := range data.ToolCalls {
		result.ToolCalls = append(result.ToolCalls, LLMToolCall{
			Name:      call.Name,
			Arguments: call.Arguments,
		})
	}
	return result
}

// LLMToolCall represents a tool call extracted from an assistant message.
type LLMToolCall struct {
	Name      string
	Arguments string
	At        time.Time
}

// Tool parameter types for parsing Arguments

// ToolParams is implemented by tool parameter types that support equality checking.
type ToolParams interface {
	Equal(other any) bool
}

// BrowserNavigateParams contains the parsed parameters for a browser_navigate tool call.
type BrowserNavigateParams struct {
	URL string `json:"url"`
}

func (p BrowserNavigateParams) Equal(other any) bool {
	if o, ok := other.(BrowserNavigateParams); ok {
		return p.URL == o.URL
	}
	return false
}

// SearchRequestParams contains the parsed parameters for a search_request tool call.
type SearchRequestParams struct {
	Query string `json:"query"`
}

func (p SearchRequestParams) Equal(other any) bool {
	if o, ok := other.(SearchRequestParams); ok {
		return p.Query == o.Query
	}
	return false
}

// BrowserNavigateParams parses and returns the parameters for a browser_navigate tool call.
// Returns nil if the tool call is not browser_navigate or if parsing fails.
func (c LLMToolCall) BrowserNavigateParams() *BrowserNavigateParams {
	if c.Name != "browser_navigate" {
		return nil
	}
	var params BrowserNavigateParams
	if err := json.Unmarshal([]byte(c.Arguments), &params); err != nil {
		return nil
	}
	return &params
}

// SearchRequestParams parses and returns the parameters for a search_request tool call.
// Returns nil if the tool call is not search_request or if parsing fails.
func (c LLMToolCall) SearchRequestParams() *SearchRequestParams {
	if c.Name != "search_request" {
		return nil
	}
	var params SearchRequestParams
	if err := json.Unmarshal([]byte(c.Arguments), &params); err != nil {
		return nil
	}
	return &params
}

// LLMToolMessageBody represents the body of a tool role message.
type LLMToolMessageBody struct {
	CallID string `json:"call_id"`
	Name   string `json:"name"`
	Output string `json:"output"`
}

// parseToolMessageBody parses a tool message body from JSON.
// Returns nil if the body cannot be parsed or doesn't contain a tool name.
func parseToolMessageBody(body json.RawMessage) *LLMToolMessageBody {
	var data LLMToolMessageBody
	if err := json.Unmarshal(body, &data); err != nil {
		return nil
	}
	if data.Name == "" {
		return nil
	}
	return &data
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

		content := parseAssistantMessageBody(msg.Body)
		if content == nil {
			continue
		}

		for i := range content.ToolCalls {
			// Set the timestamp from the parent message
			content.ToolCalls[i].At = msg.At
			calls = append(calls, content.ToolCalls[i])
		}
	}
	return calls
}

// LLMParsedMessage represents a conversation message with pre-parsed content
// ready for display in templates.
type LLMParsedMessage struct {
	Turn     int
	Role     LLMMessageRole
	At       time.Time
	Duration time.Duration

	// Content holds the parsed message content based on role.
	// For system/user: TextContent will be set
	// For tool: ToolResult will be set
	// For assistant: AssistantContent will be set
	// RawJSON is set when parsing fails or content type is unknown
	TextContent      string
	ToolResult       *LLMToolMessageBody
	AssistantContent *LLMAssistantContent
	RawJSON          string
}

// ParseMessage parses a conversation message and returns a display-ready struct.
func (msg *LLMConversationMessage) Parse() *LLMParsedMessage {
	parsed := &LLMParsedMessage{
		Turn:     msg.Turn,
		Role:     msg.Role,
		At:       msg.At,
		Duration: msg.Duration,
	}

	switch msg.Role {
	case LLMMessageRoleSystem, LLMMessageRoleUser:
		if content := parseTextContent(msg.Body); content != "" {
			parsed.TextContent = content
		} else {
			parsed.RawJSON = formatRawJSON(msg.Body)
		}
	case LLMMessageRoleTool:
		if toolInfo := parseToolMessageBody(msg.Body); toolInfo != nil {
			parsed.ToolResult = toolInfo
		} else {
			parsed.RawJSON = formatRawJSON(msg.Body)
		}
	case LLMMessageRoleAssistant:
		if content := parseAssistantMessageBody(msg.Body); content != nil {
			if len(content.ToolCalls) > 0 || content.TextOutput != "" {
				parsed.AssistantContent = content
			} else {
				parsed.RawJSON = formatRawJSON(msg.Body)
			}
		} else {
			parsed.RawJSON = formatRawJSON(msg.Body)
		}
	default:
		parsed.RawJSON = formatRawJSON(msg.Body)
	}

	return parsed
}

// ParseMessages parses all messages in a conversation for display.
func (msgs LLMConversationMessages) Parse() []*LLMParsedMessage {
	result := make([]*LLMParsedMessage, len(msgs))
	for i, msg := range msgs {
		result[i] = msg.Parse()
	}
	return result
}

// parseTextContent extracts the "content" field from a message body.
func parseTextContent(body json.RawMessage) string {
	var data struct {
		Content string `json:"content"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return ""
	}
	return data.Content
}

// formatRawJSON pretty-prints JSON for display.
func formatRawJSON(data json.RawMessage) string {
	var buf bytes.Buffer
	if err := json.Indent(&buf, data, "", "  "); err != nil {
		return string(data)
	}
	return buf.String()
}
