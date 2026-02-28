package common

// NOTE: Origin's MergeAdjacentMessages converts all message content to content-block arrays
// and does NOT preserve tool_calls as a top-level field — it folds everything into content blocks.
// Tests below are adapted to reflect origin's actual merging behaviour.

import (
	"strings"
	"testing"

	"github.com/tidwall/gjson"
)

func parseMessages(t *testing.T, raw string) []gjson.Result {
	t.Helper()
	parsed := gjson.Parse(raw)
	if !parsed.IsArray() {
		t.Fatalf("expected JSON array, got: %s", raw)
	}
	return parsed.Array()
}

// TestMergeAdjacentMessages_AssistantMergesCombinesContent verifies that two adjacent
// assistant messages are merged into one with combined text content.
func TestMergeAdjacentMessages_AssistantMergesCombinesContent(t *testing.T) {
	messages := parseMessages(t, `[
		{"role":"assistant","content":"part1"},
		{"role":"assistant","content":"part2"},
		{"role":"tool","tool_call_id":"call_1","content":"ok"}
	]`)

	merged := MergeAdjacentMessages(messages)
	if len(merged) != 2 {
		t.Fatalf("expected 2 messages after merge, got %d", len(merged))
	}

	assistant := merged[0]
	if assistant.Get("role").String() != "assistant" {
		t.Fatalf("expected first message role assistant, got %q", assistant.Get("role").String())
	}

	// Origin merges content into a combined text block.
	contentRaw := assistant.Get("content").Raw
	if !strings.Contains(contentRaw, "part1") || !strings.Contains(contentRaw, "part2") {
		t.Fatalf("expected merged content to contain both parts, got: %s", contentRaw)
	}

	if merged[1].Get("role").String() != "tool" {
		t.Fatalf("expected second message role tool, got %q", merged[1].Get("role").String())
	}
}

// TestMergeAdjacentMessages_MultipleAssistantsResultInOne verifies that two adjacent
// assistant messages are merged into a single message.
func TestMergeAdjacentMessages_MultipleAssistantsResultInOne(t *testing.T) {
	messages := parseMessages(t, `[
		{"role":"assistant","content":"first"},
		{"role":"assistant","content":"second"}
	]`)

	merged := MergeAdjacentMessages(messages)
	if len(merged) != 1 {
		t.Fatalf("expected 1 message after merge, got %d", len(merged))
	}

	if merged[0].Get("role").String() != "assistant" {
		t.Fatalf("expected merged message role assistant, got %q", merged[0].Get("role").String())
	}
}

// TestMergeAdjacentMessages_ToolMessagesRemainUnmerged verifies that tool messages
// with unique tool_call_ids are never merged together.
func TestMergeAdjacentMessages_ToolMessagesRemainUnmerged(t *testing.T) {
	messages := parseMessages(t, `[
		{"role":"tool","tool_call_id":"call_1","content":"r1"},
		{"role":"tool","tool_call_id":"call_2","content":"r2"}
	]`)

	merged := MergeAdjacentMessages(messages)
	if len(merged) != 2 {
		t.Fatalf("expected tool messages to remain separate, got %d", len(merged))
	}
}
