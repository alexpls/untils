package openai

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateRequestMarshal(t *testing.T) {
	parallel := true
	req := CreateRequest{
		Model: "grok-4-1-fast-reasoning",
		Input: []InputItem{
			{Message: &InputMessage{Role: MessageRoleSystem, Content: "be brief"}},
			{Message: &InputMessage{Role: MessageRoleUser, Content: "hello"}},
			{FunctionCall: &InputFunctionCall{CallID: "call_1", Name: "search", Arguments: `{"q":"go"}`}},
			{FunctionCallOut: &InputFunctionCallOutput{CallID: "call_1", Output: "ok"}},
		},
		Text: &TextConfig{
			Format: &JSONSchemaFormat{
				Name:   "result",
				Strict: true,
				Schema: map[string]any{"type": "object"},
			},
		},
		Tools: []Tool{
			{Name: "search", Description: "web search", Parameters: map[string]any{"type": "object"}},
		},
		ParallelToolCalls: &parallel,
	}

	buf, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(buf)

	mustContain(t, s,
		`"model":"grok-4-1-fast-reasoning"`,
		`"type":"message"`, `"role":"system"`, `"content":"be brief"`,
		`"type":"function_call"`, `"call_id":"call_1"`, `"name":"search"`, `"arguments":"{\"q\":\"go\"}"`,
		`"type":"function_call_output"`, `"output":"ok"`,
		`"text":{"format":{"type":"json_schema"`, `"name":"result"`, `"strict":true`,
		`"tools":[{"type":"function"`,
		`"parallel_tool_calls":true`,
	)
}

func TestResponseUnmarshalAndHelpers(t *testing.T) {
	raw := `{
		"output": [
			{"type": "message", "content": [
				{"type": "output_text", "text": "hello "},
				{"type": "output_text", "text": "world"}
			]},
			{"type": "function_call", "call_id": "c1", "name": "search", "arguments": "{}"}
		],
		"usage": {
			"input_tokens": 12,
			"output_tokens": 34,
			"server_side_tool_usage_details": {"web_search_calls": 2, "x_search_calls": 1}
		}
	}`

	var resp Response
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got, want := resp.OutputText(), "hello world"; got != want {
		t.Errorf("OutputText = %q, want %q", got, want)
	}
	calls := resp.FunctionCalls()
	if len(calls) != 1 || calls[0].CallID != "c1" || calls[0].Name != "search" {
		t.Errorf("FunctionCalls = %+v", calls)
	}
	if resp.Usage.InputTokens != 12 || resp.Usage.OutputTokens != 34 {
		t.Errorf("token counts wrong: %+v", resp.Usage)
	}
	extra, ok := resp.Usage.ExtraFields["server_side_tool_usage_details"]
	if !ok {
		t.Fatalf("missing extra field, got %+v", resp.Usage.ExtraFields)
	}
	var tu struct {
		Web int `json:"web_search_calls"`
		X   int `json:"x_search_calls"`
	}
	if err := json.Unmarshal(extra, &tu); err != nil {
		t.Fatalf("extra unmarshal: %v", err)
	}
	if tu.Web != 2 || tu.X != 1 {
		t.Errorf("extra fields wrong: %+v", tu)
	}
}

func TestClientErrorResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = io.WriteString(w, `{"error":{"message":"insufficient_quota"}}`)
	}))
	defer srv.Close()

	c := NewClient(WithAPIKey("x"), WithBaseURL(srv.URL))
	_, err := c.Responses.New(context.Background(), CreateRequest{Model: "m"})
	var apiErr *Error
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *Error, got %T: %v", err, err)
	}
	if apiErr.StatusCode != http.StatusTooManyRequests {
		t.Errorf("status = %d", apiErr.StatusCode)
	}
	if apiErr.Message != "insufficient_quota" {
		t.Errorf("message = %q", apiErr.Message)
	}
	if !strings.Contains(apiErr.RawJSON(), "insufficient_quota") {
		t.Errorf("raw json missing payload: %q", apiErr.RawJSON())
	}
}

func TestClientSuccessRequest(t *testing.T) {
	var gotBody []byte
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"output":[{"type":"message","content":[{"type":"output_text","text":"hi"}]}],"usage":{"input_tokens":1,"output_tokens":2}}`)
	}))
	defer srv.Close()

	c := NewClient(WithAPIKey("sk-test"), WithBaseURL(srv.URL))
	resp, err := c.Responses.New(context.Background(), CreateRequest{
		Model: "gpt-x",
		Input: []InputItem{{Message: &InputMessage{Role: MessageRoleUser, Content: "hi"}}},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if gotAuth != "Bearer sk-test" {
		t.Errorf("auth = %q", gotAuth)
	}
	if !strings.Contains(string(gotBody), `"model":"gpt-x"`) {
		t.Errorf("body = %s", gotBody)
	}
	if resp.OutputText() != "hi" {
		t.Errorf("OutputText = %q", resp.OutputText())
	}
	if resp.RawJSON() == "" {
		t.Errorf("RawJSON empty")
	}
}

func mustContain(t *testing.T, s string, subs ...string) {
	t.Helper()
	for _, sub := range subs {
		if !strings.Contains(s, sub) {
			t.Errorf("missing %q in %s", sub, s)
		}
	}
}
