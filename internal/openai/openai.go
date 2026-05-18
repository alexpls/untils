// Package openai is a minimal HTTP client for the OpenAI (and compatible, e.g.
// xAI) Responses API.
//
// It is intentionally scoped to the surface area that untils actually uses:
// non-streaming responses creation with text input, function tools, structured
// JSON-schema output, and a small slice of the response payload.
//
// The wire format mirrors the official OpenAI Responses API JSON contract so
// the same request can be sent to OpenAI- and xAI-compatible endpoints.
package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

const defaultBaseURL = "https://api.openai.com/v1"

// Client talks to the OpenAI Responses API.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client

	// Responses is the entry point for the Responses API. It mirrors the
	// official SDK shape (client.Responses.New(...)).
	Responses *ResponsesService
}

// Option configures a Client.
type Option func(*Client)

// WithAPIKey sets the API key used in the Authorization header.
func WithAPIKey(key string) Option {
	return func(c *Client) { c.apiKey = key }
}

// WithBaseURL overrides the API base URL. Useful for xAI
// ("https://api.x.ai/v1") and other OpenAI-compatible providers.
func WithBaseURL(url string) Option {
	return func(c *Client) { c.baseURL = url }
}

// WithHTTPClient overrides the underlying HTTP client (primarily for tests).
func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) { c.httpClient = h }
}

// NewClient constructs a Client. At minimum WithAPIKey should be provided.
func NewClient(opts ...Option) *Client {
	c := &Client{
		baseURL:    defaultBaseURL,
		httpClient: &http.Client{Timeout: 10 * time.Minute},
	}
	for _, opt := range opts {
		opt(c)
	}
	c.Responses = &ResponsesService{client: c}
	return c
}

// Error is returned for non-2xx HTTP responses from the API.
type Error struct {
	StatusCode int
	Message    string
	rawJSON    string
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("openai: %d %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("openai: http %d", e.StatusCode)
}

// RawJSON returns the raw response body that produced this error.
func (e *Error) RawJSON() string { return e.rawJSON }

// do performs an HTTP request and decodes the JSON body into out. On a non-2xx
// response it returns an *Error containing the raw body.
func (c *Client) do(ctx context.Context, method, path string, body any, out any, rawOut *json.RawMessage) error {
	var reqBody io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshaling request: %w", err)
		}
		reqBody = bytes.NewReader(buf)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("performing request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		apiErr := &Error{StatusCode: resp.StatusCode, rawJSON: string(respBody)}
		// Try to extract a human-readable message from the conventional
		// { "error": { "message": "..." } } envelope.
		var env struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if jerr := json.Unmarshal(respBody, &env); jerr == nil {
			apiErr.Message = env.Error.Message
		}
		return apiErr
	}

	if rawOut != nil {
		*rawOut = append((*rawOut)[:0], respBody...)
	}
	if out != nil {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("decoding response: %w", err)
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Responses API
// ---------------------------------------------------------------------------

// ResponsesService exposes the /responses endpoint.
type ResponsesService struct {
	client *Client
}

// MessageRole is the role of an input message.
type MessageRole string

const (
	MessageRoleSystem    MessageRole = "system"
	MessageRoleUser      MessageRole = "user"
	MessageRoleAssistant MessageRole = "assistant"
	MessageRoleDeveloper MessageRole = "developer"
)

// InputItem is one entry in the request's "input" array. Exactly one of the
// pointer fields should be set; the matching "type" discriminator is emitted
// automatically by MarshalJSON.
type InputItem struct {
	Message          *InputMessage
	FunctionCall     *InputFunctionCall
	FunctionCallOut  *InputFunctionCallOutput
}

// InputMessage is a plain text message from a role.
type InputMessage struct {
	Role    MessageRole `json:"role"`
	Content string      `json:"content"`
}

// InputFunctionCall echoes a prior assistant tool call back to the model.
type InputFunctionCall struct {
	CallID    string `json:"call_id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// InputFunctionCallOutput provides the output of a tool call back to the model.
type InputFunctionCallOutput struct {
	CallID string `json:"call_id"`
	Output string `json:"output"`
}

// MarshalJSON serializes an InputItem with the proper "type" tag.
func (i InputItem) MarshalJSON() ([]byte, error) {
	switch {
	case i.Message != nil:
		return json.Marshal(struct {
			Type string `json:"type"`
			*InputMessage
		}{Type: "message", InputMessage: i.Message})
	case i.FunctionCall != nil:
		return json.Marshal(struct {
			Type string `json:"type"`
			*InputFunctionCall
		}{Type: "function_call", InputFunctionCall: i.FunctionCall})
	case i.FunctionCallOut != nil:
		return json.Marshal(struct {
			Type string `json:"type"`
			*InputFunctionCallOutput
		}{Type: "function_call_output", InputFunctionCallOutput: i.FunctionCallOut})
	default:
		return nil, errors.New("openai: empty InputItem")
	}
}

// Tool is a function tool the model can call. Strict is optional; when nil it
// is omitted from the wire format (leaving the provider's default in effect).
type Tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters"`
	Strict      *bool          `json:"strict,omitempty"`
}

// MarshalJSON tags the tool as a "function" tool.
func (t Tool) MarshalJSON() ([]byte, error) {
	type alias Tool
	return json.Marshal(struct {
		Type string `json:"type"`
		alias
	}{Type: "function", alias: alias(t)})
}

// JSONSchemaFormat configures a structured JSON-schema response.
type JSONSchemaFormat struct {
	Name   string         `json:"name"`
	Strict bool           `json:"strict"`
	Schema map[string]any `json:"schema"`
}

// MarshalJSON tags the format as "json_schema".
func (f JSONSchemaFormat) MarshalJSON() ([]byte, error) {
	type alias JSONSchemaFormat
	return json.Marshal(struct {
		Type string `json:"type"`
		alias
	}{Type: "json_schema", alias: alias(f)})
}

// TextConfig configures the response text format.
type TextConfig struct {
	Format *JSONSchemaFormat `json:"format,omitempty"`
}

// CreateRequest is the body of POST /responses.
type CreateRequest struct {
	Model             string      `json:"model"`
	Input             []InputItem `json:"input"`
	Text              *TextConfig `json:"text,omitempty"`
	Tools             []Tool      `json:"tools,omitempty"`
	ParallelToolCalls *bool       `json:"parallel_tool_calls,omitempty"`
}

// Response is the parsed result from POST /responses. Unknown fields are
// preserved so callers can inspect provider-specific extensions (notably xAI's
// "server_side_tool_usage_details" inside Usage).
type Response struct {
	Output []OutputItem `json:"output"`
	Usage  Usage        `json:"usage"`

	// raw holds the verbatim JSON body for downstream consumers that want to
	// archive or re-emit the original response.
	raw json.RawMessage
}

// RawJSON returns the verbatim response body.
func (r *Response) RawJSON() string { return string(r.raw) }

// OutputText concatenates all text content from output messages, mirroring the
// official SDK's Response.OutputText() helper.
func (r *Response) OutputText() string {
	var b bytes.Buffer
	for _, item := range r.Output {
		if item.Type != "message" {
			continue
		}
		for _, c := range item.Content {
			if c.Type == "output_text" {
				b.WriteString(c.Text)
			}
		}
	}
	return b.String()
}

// FunctionCalls returns every function_call item in the output.
func (r *Response) FunctionCalls() []FunctionCall {
	out := make([]FunctionCall, 0)
	for _, item := range r.Output {
		if item.Type == "function_call" {
			out = append(out, FunctionCall{
				CallID:    item.CallID,
				Name:      item.Name,
				Arguments: item.Arguments,
			})
		}
	}
	return out
}

// OutputItem is one entry in the response's "output" array. Only the fields
// relevant to message and function_call items are decoded; other types are
// recognized by their Type discriminator and otherwise ignored.
type OutputItem struct {
	Type string `json:"type"`

	// Populated for type == "message".
	Content []OutputContent `json:"content,omitempty"`

	// Populated for type == "function_call".
	CallID    string `json:"call_id,omitempty"`
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

// OutputContent is one entry in an output message's "content" array.
type OutputContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// FunctionCall is a convenience view of a function_call output item.
type FunctionCall struct {
	CallID    string
	Name      string
	Arguments string
}

// Usage reports token usage. Unrecognized fields (e.g. xAI-specific
// "server_side_tool_usage_details") are preserved in ExtraFields as raw JSON.
type Usage struct {
	InputTokens  int64
	OutputTokens int64
	ExtraFields  map[string]json.RawMessage
}

// recognized usage field names that map to typed Usage fields.
var recognizedUsageFields = map[string]struct{}{
	"input_tokens":  {},
	"output_tokens": {},
}

// UnmarshalJSON splits Usage into typed token counts plus a bag of unknown
// fields preserved as raw JSON.
func (u *Usage) UnmarshalJSON(data []byte) error {
	raw := map[string]json.RawMessage{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if v, ok := raw["input_tokens"]; ok {
		if err := json.Unmarshal(v, &u.InputTokens); err != nil {
			return fmt.Errorf("usage.input_tokens: %w", err)
		}
	}
	if v, ok := raw["output_tokens"]; ok {
		if err := json.Unmarshal(v, &u.OutputTokens); err != nil {
			return fmt.Errorf("usage.output_tokens: %w", err)
		}
	}
	for k, v := range raw {
		if _, known := recognizedUsageFields[k]; known {
			continue
		}
		if u.ExtraFields == nil {
			u.ExtraFields = make(map[string]json.RawMessage)
		}
		u.ExtraFields[k] = v
	}
	return nil
}

// New creates a response.
func (s *ResponsesService) New(ctx context.Context, req CreateRequest) (*Response, error) {
	resp := &Response{}
	var raw json.RawMessage
	if err := s.client.do(ctx, http.MethodPost, "/responses", req, resp, &raw); err != nil {
		return nil, err
	}
	resp.raw = raw
	return resp, nil
}
