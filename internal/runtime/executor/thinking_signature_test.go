package executor

import (
	"testing"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
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

	t.Run("context_management uses clear_thinking edit", func(t *testing.T) {
		contextMgmt := map[string]interface{}{
			"edits": []map[string]interface{}{
				{
					"type": "clear_thinking_20251015",
					"keep": "all",
				},
			},
		}

		body, err := sjson.SetBytes([]byte(`{}`), "context_management", contextMgmt)
		if err != nil {
			t.Fatalf("sjson.SetBytes() error: %v", err)
		}

		editType := gjson.GetBytes(body, "context_management.edits.0.type").String()
		if editType != "clear_thinking_20251015" {
			t.Errorf("expected clear_thinking_20251015, got %q", editType)
		}

		keep := gjson.GetBytes(body, "context_management.edits.0.keep").String()
		if keep != "all" {
			t.Errorf("expected all, got %q", keep)
		}
	})

	t.Run("injectContextManagementIfThinking skips when thinking disabled", func(t *testing.T) {
		payload := []byte(`{"thinking":{"type":"disabled"},"messages":[]}`)
		result := injectContextManagementIfThinking(payload)
		if gjson.GetBytes(result, "context_management").Exists() {
			t.Error("context_management should NOT be injected when thinking is disabled")
		}
	})

	t.Run("injectContextManagementIfThinking skips when thinking absent", func(t *testing.T) {
		payload := []byte(`{"messages":[]}`)
		result := injectContextManagementIfThinking(payload)
		if gjson.GetBytes(result, "context_management").Exists() {
			t.Error("context_management should NOT be injected when thinking is absent")
		}
	})

	t.Run("injectContextManagementIfThinking injects when thinking enabled", func(t *testing.T) {
		payload := []byte(`{"thinking":{"type":"enabled","budget_tokens":8192},"messages":[]}`)
		result := injectContextManagementIfThinking(payload)
		editType := gjson.GetBytes(result, "context_management.edits.0.type").String()
		if editType != "clear_thinking_20251015" {
			t.Errorf("expected clear_thinking_20251015, got %q", editType)
		}
	})

	t.Run("injectContextManagementIfThinking injects when thinking adaptive", func(t *testing.T) {
		payload := []byte(`{"thinking":{"type":"adaptive"},"messages":[]}`)
		result := injectContextManagementIfThinking(payload)
		if !gjson.GetBytes(result, "context_management").Exists() {
			t.Error("context_management should be injected when thinking is adaptive")
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
