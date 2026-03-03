package models

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseAssistantMessageBodyNormalized(t *testing.T) {
	t.Parallel()

	body := json.RawMessage(fmt.Sprintf(`{
		"text_output": "done",
		"tool_calls": [
			{"id": "call_1", "name": "%s", "arguments": "{\"query\":\"tool\"}"}
		]
	}`, LLMToolNameSearchRequest))

	content := parseAssistantMessageBody(body)
	require.NotNil(t, content)
	require.Equal(t, "done", content.TextOutput)
	require.Len(t, content.ToolCalls, 1)
	require.Equal(t, LLMToolNameSearchRequest, content.ToolCalls[0].Name)
	require.Equal(t, "{\"query\":\"tool\"}", content.ToolCalls[0].Arguments)
}
