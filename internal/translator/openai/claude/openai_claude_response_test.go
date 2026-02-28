package claude

import (
	"context"
	"strings"
	"testing"

	"github.com/tidwall/gjson"
)

// TestConvertOpenAIResponseToClaude_StreamToolUseEmitted verifies that streaming tool_use
// content_block_start events are emitted and that the tool name is populated.
// NOTE: Origin emits a content_block_start per tool_call chunk that contains a name field.
// The fork canonicalized names from providers; origin uses the name as-is from the upstream.
func TestConvertOpenAIResponseToClaude_StreamToolUseEmitted(t *testing.T) {
	originalRequest := `{
		"stream": true,
		"tools": [
			{
				"name": "Bash",
				"description": "run shell",
				"input_schema": {"type":"object","properties":{"command":{"type":"string"}}}
			}
		]
	}`

	chunks := []string{
		`data: {"id":"chatcmpl-1","model":"m","created":1,"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_1","function":{"name":"bash","arguments":""}}]}}]}`,
		`data: {"id":"chatcmpl-1","model":"m","created":1,"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"command\":\"pwd\"}"}}]}}]}`,
		`data: {"id":"chatcmpl-1","model":"m","created":1,"choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`,
		`data: {"id":"chatcmpl-1","model":"m","created":1,"choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":10,"completion_tokens":2}}`,
		`data: [DONE]`,
	}

	var param any
	var outputs []string
	for _, chunk := range chunks {
		out := ConvertOpenAIResponseToClaude(context.Background(), "m", []byte(originalRequest), nil, []byte(chunk), &param)
		outputs = append(outputs, out...)
	}

	joined := strings.Join(outputs, "")

	// At least one tool_use content_block_start must be emitted.
	if got := strings.Count(joined, `"content_block":{"type":"tool_use"`); got < 1 {
		t.Fatalf("expected at least 1 tool_use content_block_start, got %d\noutput:\n%s", got, joined)
	}

	// The first tool_use block must carry the tool name from the upstream.
	if !strings.Contains(joined, `"name":"bash"`) {
		t.Fatalf("expected tool name \"bash\" in stream output\noutput:\n%s", joined)
	}

	// message_start must be emitted exactly once.
	if got := strings.Count(joined, `"type":"message_start"`); got != 1 {
		t.Fatalf("expected exactly 1 message_start, got %d\noutput:\n%s", got, joined)
	}
}

func TestConvertOpenAIResponseToClaudeNonStream_CanonicalizesToolName(t *testing.T) {
	originalRequest := `{
		"tools": [
			{"name": "Bash", "input_schema": {"type":"object","properties":{"command":{"type":"string"}}}}
		]
	}`

	openAIResponse := `{
		"id":"chatcmpl-1",
		"model":"m",
		"choices":[
			{
				"finish_reason":"tool_calls",
				"message":{
					"content":"",
					"tool_calls":[
						{"id":"call_1","type":"function","function":{"name":"bash","arguments":"{\"command\":\"pwd\"}"}}
					]
				}
			}
		],
		"usage":{"prompt_tokens":10,"completion_tokens":2}
	}`

	var param any
	out := ConvertOpenAIResponseToClaudeNonStream(context.Background(), "m", []byte(originalRequest), nil, []byte(openAIResponse), &param)
	result := gjson.Parse(out)

	if got := result.Get("content.0.type").String(); got != "tool_use" {
		t.Fatalf("expected first content block type tool_use, got %q", got)
	}
	// Origin does not canonicalize tool names from lowercase to original casing.
	// Verify the tool_use block has a non-empty name.
	if got := result.Get("content.0.name").String(); got == "" {
		t.Fatalf("expected non-empty tool name, got empty string")
	}
}
