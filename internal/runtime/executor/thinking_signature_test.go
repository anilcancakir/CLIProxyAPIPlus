package executor

import (
	"testing"

	"github.com/tidwall/gjson"
)

func TestThinkingSignature(t *testing.T) {
	t.Run("injectMessagesCacheControl does not mutate thinking block", func(t *testing.T) {
		payload := []byte(`{
  "messages": [
    {"role":"user","content":[{"type":"thinking","thinking":"...","signature":"abc123"}]},
    {"role":"assistant","content":[{"type":"text","text":"response"}]},
    {"role":"user","content":[{"type":"text","text":"follow-up"}]}
  ]
}`)

		result := injectMessagesCacheControl(payload)

		cc := gjson.GetBytes(result, "messages.0.content.0.cache_control")
		if cc.Exists() {
			t.Error("cache_control was added to thinking block — signature would be invalidated")
		}
	})

	t.Run("injectMessagesCacheControl still mutates text block", func(t *testing.T) {
		payload := []byte(`{
  "messages": [
    {"role":"user","content":[{"type":"text","text":"first user message"}]},
    {"role":"assistant","content":[{"type":"text","text":"response"}]},
    {"role":"user","content":[{"type":"text","text":"second user message"}]}
  ]
}`)

		result := injectMessagesCacheControl(payload)

		cc := gjson.GetBytes(result, "messages.0.content.0.cache_control")
		if !cc.Exists() {
			t.Error("text block should get cache_control")
		}
	})

	t.Run("normalizeCacheControlTTL preserves thinking block bytes", func(t *testing.T) {
		// Payload with a thinking block containing a signature — the exact bytes must survive.
		payload := []byte(`{"system":[{"type":"text","text":"hello","cache_control":{"type":"ephemeral"}}],"messages":[{"role":"assistant","content":[{"type":"thinking","thinking":"deep thought","signature":"sig_abc123_very_long_signature_value_here_1234567890"}]},{"role":"user","content":[{"type":"text","text":"hi"}]}]}`)

		// Extract original thinking block.
		origSig := gjson.GetBytes(payload, "messages.0.content.0.signature").String()
		origThinking := gjson.GetBytes(payload, "messages.0.content.0.thinking").String()

		// Run normalization — should NOT round-trip through json.Marshal.
		result := normalizeCacheControlTTL(payload)

		newSig := gjson.GetBytes(result, "messages.0.content.0.signature").String()
		newThinking := gjson.GetBytes(result, "messages.0.content.0.thinking").String()
		if newSig != origSig {
			t.Errorf("signature changed: %q -> %q", origSig, newSig)
		}
		if newThinking != origThinking {
			t.Errorf("thinking changed: %q -> %q", origThinking, newThinking)
		}
	})

	t.Run("enforceCacheControlLimit preserves thinking block bytes", func(t *testing.T) {
		// 5 cache_control blocks — will trigger enforcement (max=4).
		payload := []byte(`{
		"system":[
			{"type":"text","text":"s1","cache_control":{"type":"ephemeral"}},
			{"type":"text","text":"s2","cache_control":{"type":"ephemeral"}}
		],
		"tools":[
			{"name":"t1","cache_control":{"type":"ephemeral"}},
			{"name":"t2","cache_control":{"type":"ephemeral"}}
		],
		"messages":[
			{"role":"assistant","content":[{"type":"thinking","thinking":"deep","signature":"sig_xyz_long_enough_for_validation_check_1234567890"}]},
			{"role":"user","content":[{"type":"text","text":"hi","cache_control":{"type":"ephemeral"}}]}
		]
		}`)

		origSig := gjson.GetBytes(payload, "messages.0.content.0.signature").String()
		result := enforceCacheControlLimit(payload, 4)
		newSig := gjson.GetBytes(result, "messages.0.content.0.signature").String()
		if newSig != origSig {
			t.Errorf("signature changed after enforceCacheControlLimit: %q -> %q", origSig, newSig)
		}
	})
}

func TestStripThinkingBlocks(t *testing.T) {
	t.Run("strips thinking blocks from assistant messages", func(t *testing.T) {
		payload := []byte(`{"messages":[
			{"role":"user","content":[{"type":"text","text":"hello"}]},
			{"role":"assistant","content":[
				{"type":"thinking","thinking":"deep thought","signature":"sig123"},
				{"type":"text","text":"response"}
			]},
			{"role":"user","content":[{"type":"text","text":"follow-up"}]}
		]}`)

		result := stripThinkingBlocks(payload)

		// Thinking block should be gone.
		thinking := gjson.GetBytes(result, "messages.1.content.0.type").String()
		if thinking == "thinking" {
			t.Error("thinking block should have been stripped")
		}

		// Text block should remain.
		text := gjson.GetBytes(result, "messages.1.content.0.type").String()
		if text != "text" {
			t.Errorf("expected text block to remain, got %q", text)
		}

		// Message count should be preserved.
		msgCount := gjson.GetBytes(result, "messages.#").Int()
		if msgCount != 3 {
			t.Errorf("expected 3 messages, got %d", msgCount)
		}
	})

	t.Run("removes empty assistant message after stripping", func(t *testing.T) {
		payload := []byte(`{"messages":[
			{"role":"user","content":[{"type":"text","text":"hello"}]},
			{"role":"assistant","content":[
				{"type":"thinking","thinking":"deep thought","signature":"sig123"}
			]},
			{"role":"user","content":[{"type":"text","text":"follow-up"}]}
		]}`)

		result := stripThinkingBlocks(payload)

		// The assistant message with only thinking should be removed entirely.
		msgCount := gjson.GetBytes(result, "messages.#").Int()
		if msgCount != 2 {
			t.Errorf("expected 2 messages (thinking-only assistant removed), got %d", msgCount)
		}
	})

	t.Run("strips redacted_thinking blocks", func(t *testing.T) {
		payload := []byte(`{"messages":[
			{"role":"user","content":[{"type":"text","text":"hello"}]},
			{"role":"assistant","content":[
				{"type":"redacted_thinking","data":"abc"},
				{"type":"text","text":"response"}
			]}
		]}`)

		result := stripThinkingBlocks(payload)

		contentCount := gjson.GetBytes(result, "messages.1.content.#").Int()
		if contentCount != 1 {
			t.Errorf("expected 1 content block after strip, got %d", contentCount)
		}
		remaining := gjson.GetBytes(result, "messages.1.content.0.type").String()
		if remaining != "text" {
			t.Errorf("expected text block to remain, got %q", remaining)
		}
	})

	t.Run("does not touch user messages", func(t *testing.T) {
		payload := []byte(`{"messages":[
			{"role":"user","content":[{"type":"text","text":"hello"}]},
			{"role":"assistant","content":[{"type":"text","text":"response"}]}
		]}`)

		result := stripThinkingBlocks(payload)

		// Should be unchanged.
		msgCount := gjson.GetBytes(result, "messages.#").Int()
		if msgCount != 2 {
			t.Errorf("expected 2 messages, got %d", msgCount)
		}
	})

	t.Run("no-op when no thinking blocks present", func(t *testing.T) {
		payload := []byte(`{"messages":[
			{"role":"user","content":[{"type":"text","text":"hello"}]},
			{"role":"assistant","content":[{"type":"text","text":"response"}]}
		]}`)

		result := stripThinkingBlocks(payload)

		if string(result) != string(payload) {
			t.Error("payload should be unchanged when no thinking blocks present")
		}
	})

	t.Run("handles multiple assistant messages with thinking", func(t *testing.T) {
		payload := []byte(`{"messages":[
			{"role":"user","content":[{"type":"text","text":"q1"}]},
			{"role":"assistant","content":[
				{"type":"thinking","thinking":"thought 1","signature":"sig1"},
				{"type":"text","text":"a1"}
			]},
			{"role":"user","content":[{"type":"text","text":"q2"}]},
			{"role":"assistant","content":[
				{"type":"thinking","thinking":"thought 2","signature":"sig2"},
				{"type":"text","text":"a2"}
			]}
		]}`)

		result := stripThinkingBlocks(payload)

		// Both assistant messages should have thinking stripped but text preserved.
		msgCount := gjson.GetBytes(result, "messages.#").Int()
		if msgCount != 4 {
			t.Errorf("expected 4 messages, got %d", msgCount)
		}

		a1Content := gjson.GetBytes(result, "messages.1.content.#").Int()
		if a1Content != 1 {
			t.Errorf("first assistant should have 1 content block, got %d", a1Content)
		}

		a2Content := gjson.GetBytes(result, "messages.3.content.#").Int()
		if a2Content != 1 {
			t.Errorf("second assistant should have 1 content block, got %d", a2Content)
		}
	})

	t.Run("preserves tool_use blocks alongside thinking strip", func(t *testing.T) {
		payload := []byte(`{"messages":[
			{"role":"user","content":[{"type":"text","text":"hello"}]},
			{"role":"assistant","content":[
				{"type":"thinking","thinking":"let me use a tool","signature":"sig_tool"},
				{"type":"tool_use","id":"t1","name":"read","input":{"path":"a.go"}}
			]},
			{"role":"user","content":[{"type":"tool_result","tool_use_id":"t1","content":"file content"}]}
		]}`)

		result := stripThinkingBlocks(payload)

		// tool_use should remain.
		toolType := gjson.GetBytes(result, "messages.1.content.0.type").String()
		if toolType != "tool_use" {
			t.Errorf("expected tool_use to remain, got %q", toolType)
		}

		// Only 1 content block (thinking stripped).
		contentCount := gjson.GetBytes(result, "messages.1.content.#").Int()
		if contentCount != 1 {
			t.Errorf("expected 1 content block after strip, got %d", contentCount)
		}
	})
}
