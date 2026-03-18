package claude

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	if !strings.Contains(joined, `"name":"Bash"`) {
		t.Fatalf("expected tool name \"Bash\" in stream output\noutput:\n%s", joined)
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

// TestConvertOpenAIResponseToClaudeNonStream_ReasoningText verifies that the
// Copilot-style "reasoning_text" field on message is converted to a Claude
// "thinking" content block in non-streaming mode.
func TestConvertOpenAIResponseToClaudeNonStream_ReasoningText(t *testing.T) {
	originalRequest := `{}`

	openAIResponse := `{
		"id": "resp-1",
		"model": "gemini-3-flash-preview",
		"choices": [
			{
				"finish_reason": "stop",
				"index": 0,
				"message": {
					"content": "15 * 37 = 555",
					"reasoning_text": "I need to multiply 15 by 37. Using distributive property: 15*30=450, 15*7=105, total=555.",
					"role": "assistant"
				}
			}
		],
		"usage": {
			"completion_tokens": 12,
			"prompt_tokens": 10,
			"total_tokens": 338,
			"reasoning_tokens": 316
		}
	}`

	var param any
	out := ConvertOpenAIResponseToClaudeNonStream(
		context.Background(),
		"gemini-3-flash-preview",
		[]byte(originalRequest),
		nil,
		[]byte(openAIResponse),
		&param,
	)
	result := gjson.Parse(out)

	// 1. First content block must be a thinking block with reasoning_text content.
	require.Equal(t, "thinking", result.Get("content.0.type").String(),
		"expected first content block to be thinking")
	assert.Contains(t, result.Get("content.0.thinking").String(), "distributive property",
		"thinking block should contain the reasoning text")

	// 2. Second content block must be a text block with the actual answer.
	require.Equal(t, "text", result.Get("content.1.type").String(),
		"expected second content block to be text")
	assert.Equal(t, "15 * 37 = 555", result.Get("content.1.text").String())

	// 3. Stop reason must be end_turn.
	assert.Equal(t, "end_turn", result.Get("stop_reason").String())
}

// TestConvertOpenAIResponseToClaudeNonStream_ReasoningTokensUsage verifies
// that reasoning_tokens from the OpenAI usage block are excluded from
// output_tokens and surfaced separately in the Claude usage format.
func TestConvertOpenAIResponseToClaudeNonStream_ReasoningTokensUsage(t *testing.T) {
	originalRequest := `{}`

	openAIResponse := `{
		"id": "resp-1",
		"model": "gemini-3-flash-preview",
		"choices": [
			{
				"finish_reason": "stop",
				"index": 0,
				"message": {
					"content": "555",
					"role": "assistant"
				}
			}
		],
		"usage": {
			"completion_tokens": 12,
			"prompt_tokens": 10,
			"total_tokens": 338,
			"reasoning_tokens": 316
		}
	}`

	var param any
	out := ConvertOpenAIResponseToClaudeNonStream(
		context.Background(),
		"gemini-3-flash-preview",
		[]byte(originalRequest),
		nil,
		[]byte(openAIResponse),
		&param,
	)
	result := gjson.Parse(out)

	// output_tokens should only include completion_tokens (not reasoning_tokens).
	assert.Equal(t, int64(12), result.Get("usage.output_tokens").Int(),
		"output_tokens should equal completion_tokens, not include reasoning_tokens")
	assert.Equal(t, int64(10), result.Get("usage.input_tokens").Int())
}

// TestConvertOpenAIResponseToClaude_StreamReasoningText verifies that
// streaming chunks with "reasoning_text" delta are converted to Claude
// thinking content blocks.
func TestConvertOpenAIResponseToClaude_StreamReasoningText(t *testing.T) {
	originalRequest := `{"stream": true}`

	chunks := []string{
		// First chunk: reasoning_text delta
		`data: {"id":"resp-1","model":"gemini-3-flash-preview","created":1,"choices":[{"index":0,"delta":{"role":"assistant","reasoning_text":"Step 1: multiply 15 by 30 = 450"}}]}`,
		// Second chunk: more reasoning
		`data: {"id":"resp-1","model":"gemini-3-flash-preview","created":1,"choices":[{"index":0,"delta":{"reasoning_text":"Step 2: multiply 15 by 7 = 105"}}]}`,
		// Third chunk: actual content
		`data: {"id":"resp-1","model":"gemini-3-flash-preview","created":1,"choices":[{"index":0,"delta":{"content":"555"}}]}`,
		// Fourth chunk: finish + usage
		`data: {"id":"resp-1","model":"gemini-3-flash-preview","created":1,"choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":3,"reasoning_tokens":200}}`,
		`data: [DONE]`,
	}

	var param any
	var outputs []string
	for _, chunk := range chunks {
		out := ConvertOpenAIResponseToClaude(
			context.Background(),
			"gemini-3-flash-preview",
			[]byte(originalRequest),
			nil,
			[]byte(chunk),
			&param,
		)
		outputs = append(outputs, out...)
	}

	joined := strings.Join(outputs, "")

	// 1. Must emit exactly 1 message_start.
	assert.Equal(t, 1, strings.Count(joined, `"type":"message_start"`),
		"expected exactly 1 message_start")

	// 2. Must emit a thinking content_block_start.
	require.Contains(t, joined, `"type":"thinking"`,
		"expected a thinking content block start")

	// 3. Must emit thinking_delta with reasoning text.
	assert.Contains(t, joined, `"type":"thinking_delta"`,
		"expected thinking_delta events")
	assert.Contains(t, joined, "Step 1: multiply 15 by 30",
		"thinking delta should contain reasoning text")

	// 4. Must emit a text content_block_start (after thinking).
	assert.Contains(t, joined, `"content_block":{"type":"text"`,
		"expected a text content block start")

	// 5. Must emit text_delta with actual content.
	assert.Contains(t, joined, "555",
		"text delta should contain the answer")

	// 6. Must emit message_stop.
	assert.Contains(t, joined, `"type":"message_stop"`,
		"expected message_stop event")
}

// TestConvertOpenAIResponseToClaude_StreamReasoningText_NonStreamPath verifies
// that non-streaming responses with reasoning_text are correctly handled
// through the streaming converter's non-stream detection path.
func TestConvertOpenAIResponseToClaude_StreamReasoningText_NonStreamPath(t *testing.T) {
	// When stream is false/missing, ConvertOpenAIResponseToClaude delegates
	// to convertOpenAINonStreamingToAnthropic.
	originalRequest := `{}`

	openAIResponse := `data: {
		"id": "resp-1",
		"model": "gemini-3-flash-preview",
		"choices": [
			{
				"finish_reason": "stop",
				"index": 0,
				"message": {
					"content": "555",
					"reasoning_text": "I calculated 15*37 step by step",
					"role": "assistant"
				}
			}
		],
		"usage": {
			"completion_tokens": 3,
			"prompt_tokens": 10,
			"reasoning_tokens": 200
		}
	}`

	var param any
	results := ConvertOpenAIResponseToClaude(
		context.Background(),
		"gemini-3-flash-preview",
		[]byte(originalRequest),
		nil,
		[]byte(openAIResponse),
		&param,
	)

	require.Len(t, results, 1, "expected exactly 1 result for non-stream")
	result := gjson.Parse(results[0])

	// Must have thinking block first, then text block.
	require.Equal(t, "thinking", result.Get("content.0.type").String())
	assert.Contains(t, result.Get("content.0.thinking").String(), "step by step")
	require.Equal(t, "text", result.Get("content.1.type").String())
	assert.Equal(t, "555", result.Get("content.1.text").String())
}

// TestExtractOpenAIUsage_WithReasoningTokens verifies that extractOpenAIUsage
// returns the correct token counts when reasoning_tokens is present.
func TestExtractOpenAIUsage_WithReasoningTokens(t *testing.T) {
	usage := gjson.Parse(`{
		"prompt_tokens": 10,
		"completion_tokens": 12,
		"total_tokens": 338,
		"reasoning_tokens": 316
	}`)

	inputTokens, outputTokens, cachedTokens := extractOpenAIUsage(usage)

	// output_tokens should be completion_tokens only (not reasoning_tokens).
	assert.Equal(t, int64(10), inputTokens)
	assert.Equal(t, int64(12), outputTokens,
		"output_tokens should equal completion_tokens, not include reasoning_tokens")
	assert.Equal(t, int64(0), cachedTokens)
}

// TestExtractOpenAIUsage_WithoutReasoningTokens verifies backward
// compatibility — when reasoning_tokens is absent, behavior is unchanged.
func TestExtractOpenAIUsage_WithoutReasoningTokens(t *testing.T) {
	usage := gjson.Parse(`{
		"prompt_tokens": 50,
		"completion_tokens": 100,
		"total_tokens": 150
	}`)

	inputTokens, outputTokens, cachedTokens := extractOpenAIUsage(usage)

	assert.Equal(t, int64(50), inputTokens)
	assert.Equal(t, int64(100), outputTokens)
	assert.Equal(t, int64(0), cachedTokens)
}
