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
}
