package executor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v6/internal/config"
)

// TestApplySystemPromptRules_EmptyConfig tests that empty config returns payload unchanged.
func TestApplySystemPromptRules_EmptyConfig(t *testing.T) {
	cfg := &config.Config{}
	payload := []byte(`{"model":"claude-opus-4","messages":[]}`)

	result := applySystemPromptRules(cfg, "claude-opus-4", "claude", payload, "")

	if string(result) != string(payload) {
		t.Errorf("Expected payload unchanged, got %s", string(result))
	}
}

func TestApplySystemPromptRules_NoMatchingRules(t *testing.T) {
	cfg := &config.Config{
		Payload: config.PayloadConfig{
			SystemPrompts: []config.SystemPromptRule{
				{
					Models: []config.PayloadModelRule{
						{Name: "gpt-*"},
					},
					Prompt: "GPT prompt",
					Mode:   "prepend",
				},
			},
		},
	}
	payload := []byte(`{"model":"claude-opus-4","system":[{"type":"text","text":"original"}]}`)

	result := applySystemPromptRules(cfg, "claude-opus-4", "claude", payload, "")

	if string(result) != string(payload) {
		t.Errorf("Expected payload unchanged for non-matching model, got %s", string(result))
	}
}

func TestApplySystemPromptRules_MatchingModel(t *testing.T) {
	cfg := &config.Config{
		Payload: config.PayloadConfig{
			SystemPrompts: []config.SystemPromptRule{
				{
					Models: []config.PayloadModelRule{
						{Name: "claude-*"},
					},
					Prompt: "Injected prompt",
					Mode:   "prepend",
				},
			},
		},
	}
	payload := []byte(`{"model":"claude-opus-4","system":[{"type":"text","text":"original"}]}`)

	result := applySystemPromptRules(cfg, "claude-opus-4", "claude", payload, "")

	if string(result) == string(payload) {
		t.Error("Expected payload to be modified for matching model")
	}
}

// TestInjectClaudeSystemPrompt_Prepend tests prepend mode for Claude format.
func TestInjectClaudeSystemPrompt_Prepend(t *testing.T) {
	payload := []byte(`{"system":[{"type":"text","text":"original"}]}`)
	prompt := "Prepended prompt"

	result := injectClaudeSystemPrompt(payload, prompt, "prepend")

	expected := `"text":"Prepended prompt"`
	if !contains(string(result), expected) {
		t.Errorf("Expected result to contain %q, got %s", expected, string(result))
	}
}

// TestInjectClaudeSystemPrompt_Replace tests replace mode for Claude format.
func TestInjectClaudeSystemPrompt_Replace(t *testing.T) {
	payload := []byte(`{"system":[{"type":"text","text":"original"}]}`)
	prompt := "Replaced prompt"

	result := injectClaudeSystemPrompt(payload, prompt, "replace")

	if contains(string(result), `"text":"original"`) {
		t.Error("Expected original prompt to be replaced")
	}
	if !contains(string(result), `"text":"Replaced prompt"`) {
		t.Errorf("Expected replaced prompt, got %s", string(result))
	}
}

// TestInjectClaudeSystemPrompt_NoExistingSystem tests injection when no system field exists.
func TestInjectClaudeSystemPrompt_NoExistingSystem(t *testing.T) {
	payload := []byte(`{"model":"claude-opus-4"}`)
	prompt := "New system prompt"

	result := injectClaudeSystemPrompt(payload, prompt, "prepend")

	if !contains(string(result), `"system"`) {
		t.Error("Expected system field to be created")
	}
	if !contains(string(result), `"text":"New system prompt"`) {
		t.Errorf("Expected new prompt, got %s", string(result))
	}
}

// TestInjectOpenAISystemPrompt_Prepend tests prepend mode for OpenAI format.
func TestInjectOpenAISystemPrompt_Prepend(t *testing.T) {
	payload := []byte(`{"messages":[{"role":"user","content":"Hello"}]}`)
	prompt := "System instruction"

	result := injectOpenAISystemPrompt(payload, prompt, "prepend")

	expectedRole := `"role":"system"`
	if !contains(string(result), expectedRole) {
		t.Errorf("Expected system message, got %s", string(result))
	}
	expectedContent := `"content":"System instruction"`
	if !contains(string(result), expectedContent) {
		t.Errorf("Expected system content, got %s", string(result))
	}
}

// TestInjectOpenAISystemPrompt_Replace tests replace mode for OpenAI format.
func TestInjectOpenAISystemPrompt_Replace(t *testing.T) {
	payload := []byte(`{"messages":[{"role":"system","content":"old"},{"role":"user","content":"Hello"}]}`)
	prompt := "New system"

	result := injectOpenAISystemPrompt(payload, prompt, "replace")

	if contains(string(result), `"content":"old"`) {
		t.Error("Expected old system message to be removed")
	}
	if !contains(string(result), `"content":"New system"`) {
		t.Errorf("Expected new system message, got %s", string(result))
	}
}

// TestInjectOpenAISystemPrompt_NoMessages tests injection when no messages field exists.
func TestInjectOpenAISystemPrompt_NoMessages(t *testing.T) {
	payload := []byte(`{"model":"gpt-4"}`)
	prompt := "System prompt"

	result := injectOpenAISystemPrompt(payload, prompt, "prepend")

	if !contains(string(result), `"messages"`) {
		t.Error("Expected messages field to be created")
	}
}

// TestApplySystemPromptRules_ProtocolFilter tests protocol filtering.
func TestApplySystemPromptRules_ProtocolFilter(t *testing.T) {
	cfg := &config.Config{
		Payload: config.PayloadConfig{
			SystemPrompts: []config.SystemPromptRule{
				{
					Models: []config.PayloadModelRule{
						{Name: "claude-*", Protocol: "claude"},
					},
					Prompt: "Claude only",
					Mode:   "prepend",
				},
			},
		},
	}
	payload := []byte(`{"system":[]}`)

	// Should match when protocol is "claude"
	result := applySystemPromptRules(cfg, "claude-opus-4", "claude", payload, "")
	if string(result) == string(payload) {
		t.Error("Expected payload to be modified for matching protocol")
	}

	payload2 := []byte(`{"system":[]}`)
	result2 := applySystemPromptRules(cfg, "claude-opus-4", "openai", payload2, "")
	if string(result2) != string(payload2) {
		t.Error("Expected payload unchanged for non-matching protocol")
	}
}

// TestApplySystemPromptRules_MultipleRules tests multiple rules applied in order.
func TestApplySystemPromptRules_MultipleRules(t *testing.T) {
	cfg := &config.Config{
		Payload: config.PayloadConfig{
			SystemPrompts: []config.SystemPromptRule{
				{
					Models: []config.PayloadModelRule{
						{Name: "claude-opus-*"},
					},
					Prompt: "Opus specific",
					Mode:   "prepend",
				},
				{
					Models: []config.PayloadModelRule{
						{Name: "claude-*"},
					},
					Prompt: "General Claude",
					Mode:   "append",
				},
			},
		},
	}
	payload := []byte(`{"system":[]}`)

	result := applySystemPromptRules(cfg, "claude-opus-4", "claude", payload, "")

	// Both rules should apply
	if !contains(string(result), "Opus specific") {
		t.Error("Expected first rule to apply")
	}
	if !contains(string(result), "General Claude") {
		t.Error("Expected second rule to apply")
	}
}

// TestApplySystemPromptRules_WithPromptFile tests loading prompt from file.
func TestApplySystemPromptRules_WithPromptFile(t *testing.T) {
	// Create a temporary prompt file
	tmpDir := t.TempDir()
	promptFile := filepath.Join(tmpDir, "test-prompt.md")
	promptContent := "You are Can, a helpful AI assistant from file."

	if err := os.WriteFile(promptFile, []byte(promptContent), 0644); err != nil {
		t.Fatalf("Failed to create test prompt file: %v", err)
	}

	cfg := &config.Config{
		Payload: config.PayloadConfig{
			SystemPrompts: []config.SystemPromptRule{
				{
					Models: []config.PayloadModelRule{
						{Name: "can"},
					},
					Prompt:     "Inline prompt", // Should be overridden by file
					PromptFile: promptFile,
					Mode:       "prepend",
				},
			},
		},
	}

	// Sanitize to load file content
	cfg.SanitizeSystemPromptRules(tmpDir)

	payload := []byte(`{"messages":[]}`)
	result := applySystemPromptRules(cfg, "can", "openai", payload, "can")

	// Should use file content, not inline
	if !contains(string(result), promptContent) {
		t.Errorf("Expected prompt from file, got %s", string(result))
	}
	if contains(string(result), "Inline prompt") {
		t.Error("Should not contain inline prompt when file is specified")
	}
}

// TestApplySystemPromptRules_PromptFileNotFound tests handling of missing prompt file.
func TestApplySystemPromptRules_PromptFileNotFound(t *testing.T) {
	cfg := &config.Config{
		Payload: config.PayloadConfig{
			SystemPrompts: []config.SystemPromptRule{
				{
					Models: []config.PayloadModelRule{
						{Name: "can"},
					},
					PromptFile: "/nonexistent/path/prompt.md",
					Mode:       "prepend",
				},
			},
		},
	}

	// Sanitize should drop the invalid rule
	cfg.SanitizeSystemPromptRules("")

	// Rule should be dropped (empty SystemPrompts)
	if len(cfg.Payload.SystemPrompts) != 0 {
		t.Error("Expected rule to be dropped when prompt file not found")
	}
}

// TestApplySystemPromptRules_PromptFileEmpty tests handling of empty prompt file.
func TestApplySystemPromptRules_PromptFileEmpty(t *testing.T) {
	// Create an empty temporary prompt file
	tmpDir := t.TempDir()
	promptFile := filepath.Join(tmpDir, "empty-prompt.md")

	if err := os.WriteFile(promptFile, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to create empty test prompt file: %v", err)
	}

	cfg := &config.Config{
		Payload: config.PayloadConfig{
			SystemPrompts: []config.SystemPromptRule{
				{
					Models: []config.PayloadModelRule{
						{Name: "can"},
					},
					PromptFile: promptFile,
					Mode:       "prepend",
				},
			},
		},
	}

	// Sanitize should drop the invalid rule
	cfg.SanitizeSystemPromptRules(tmpDir)

	// Rule should be dropped
	if len(cfg.Payload.SystemPrompts) != 0 {
		t.Error("Expected rule to be dropped when prompt file is empty")
	}
}

// TestApplySystemPromptRules_PromptFileRelativePath tests relative path resolution.
func TestApplySystemPromptRules_PromptFileRelativePath(t *testing.T) {
	// Create a temporary directory structure
	tmpDir := t.TempDir()
	promptsDir := filepath.Join(tmpDir, "prompts")
	if err := os.MkdirAll(promptsDir, 0755); err != nil {
		t.Fatalf("Failed to create prompts dir: %v", err)
	}

	promptFile := filepath.Join(promptsDir, "can.md")
	promptContent := "Can assistant from relative path."

	if err := os.WriteFile(promptFile, []byte(promptContent), 0644); err != nil {
		t.Fatalf("Failed to create test prompt file: %v", err)
	}

	cfg := &config.Config{
		Payload: config.PayloadConfig{
			SystemPrompts: []config.SystemPromptRule{
				{
					Models: []config.PayloadModelRule{
						{Name: "can"},
					},
					PromptFile: "prompts/can.md", // Relative path
					Mode:       "prepend",
				},
			},
		},
	}

	// Sanitize with config directory
	cfg.SanitizeSystemPromptRules(tmpDir)

	// Rule should be valid and contain file content
	if len(cfg.Payload.SystemPrompts) != 1 {
		t.Fatal("Expected rule to be valid")
	}

	if cfg.Payload.SystemPrompts[0].Prompt != promptContent {
		t.Errorf("Expected prompt %q, got %q", promptContent, cfg.Payload.SystemPrompts[0].Prompt)
	}
}

// TestApplySystemPromptRules_PromptFileMarkdown tests loading from .md file.
func TestApplySystemPromptRules_PromptFileMarkdown(t *testing.T) {
	tmpDir := t.TempDir()
	promptFile := filepath.Join(tmpDir, "system-prompt.md")
	// Markdown content with formatting
	promptContent := `# System Prompt

You are **Can**, a helpful AI assistant.

## Guidelines:
- Be concise
- Be helpful
- Be accurate`

	if err := os.WriteFile(promptFile, []byte(promptContent), 0644); err != nil {
		t.Fatalf("Failed to create test prompt file: %v", err)
	}

	cfg := &config.Config{
		Payload: config.PayloadConfig{
			SystemPrompts: []config.SystemPromptRule{
				{
					Models: []config.PayloadModelRule{
						{Name: "can"},
					},
					PromptFile: promptFile,
					Mode:       "prepend",
				},
			},
		},
	}

	cfg.SanitizeSystemPromptRules(tmpDir)

	payload := []byte(`{"messages":[{"role":"user","content":"Hello"}]}`)
	result := applySystemPromptRules(cfg, "can", "openai", payload, "can")

	// Should contain the markdown content
	if !contains(string(result), "You are **Can**") {
		t.Errorf("Expected markdown content in result, got %s", string(result))
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || containsInternal(s, substr))
}

func containsInternal(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
