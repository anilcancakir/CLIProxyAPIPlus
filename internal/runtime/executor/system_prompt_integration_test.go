package executor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v6/internal/config"
)

// TestSystemPromptIntegration_EndToEnd tests the full flow with file-based prompt
func TestSystemPromptIntegration_EndToEnd(t *testing.T) {
	// Create temp directory with prompt file
	tmpDir := t.TempDir()
	promptFile := filepath.Join(tmpDir, "can.md")
	promptContent := "You are Can, an AI assistant provided by Kodizm."

	if err := os.WriteFile(promptFile, []byte(promptContent), 0644); err != nil {
		t.Fatalf("Failed to create prompt file: %v", err)
	}

	// Create config matching the server setup
	cfg := &config.Config{
		Payload: config.PayloadConfig{
			SystemPrompts: []config.SystemPromptRule{
				{
					Models: []config.PayloadModelRule{
						{
							Name:     "can",
							Protocol: "openai",
						},
					},
					PromptFile: "./can.md",
					Mode:       "prepend",
				},
			},
		},
	}

	// Sanitize config (this loads the file)
	cfg.SanitizeSystemPromptRules(tmpDir)

	// Verify rule is loaded
	if len(cfg.Payload.SystemPrompts) != 1 {
		t.Fatalf("Expected 1 rule after sanitization, got %d", len(cfg.Payload.SystemPrompts))
	}

	// Verify prompt content is loaded
	if cfg.Payload.SystemPrompts[0].Prompt != promptContent {
		t.Errorf("Expected prompt %q, got %q", promptContent, cfg.Payload.SystemPrompts[0].Prompt)
	}

	// Simulate the request flow
	payload := []byte(`{"model":"can","messages":[{"role":"user","content":"Who are you?"}]}`)

	// Call applySystemPromptRules like the executor does
	result := applySystemPromptRules(cfg, "can", "openai", payload, "can")

	// Verify the system prompt was injected
	resultStr := string(result)
	if !contains(resultStr, "You are Can") {
		t.Errorf("Expected system prompt in result, got: %s", resultStr)
	}

	t.Logf("Success! Result: %s", resultStr)
}

// TestSystemPromptIntegration_WithOpenAIFormat tests with actual OpenAI format payload
func TestSystemPromptIntegration_WithOpenAIFormat(t *testing.T) {
	tmpDir := t.TempDir()
	promptFile := filepath.Join(tmpDir, "test.md")
	promptContent := "You are TestAI, a helpful assistant."

	if err := os.WriteFile(promptFile, []byte(promptContent), 0644); err != nil {
		t.Fatalf("Failed to create prompt file: %v", err)
	}

	cfg := &config.Config{
		Payload: config.PayloadConfig{
			SystemPrompts: []config.SystemPromptRule{
				{
					Models: []config.PayloadModelRule{
						{Name: "test-model", Protocol: "openai"},
					},
					PromptFile: "./test.md",
					Mode:       "prepend",
				},
			},
		},
	}

	cfg.SanitizeSystemPromptRules(tmpDir)

	// Test with no existing messages
	payload1 := []byte(`{"model":"test-model"}`)
	result1 := applySystemPromptRules(cfg, "test-model", "openai", payload1, "test-model")
	t.Logf("Result with no messages: %s", string(result1))

	if !contains(string(result1), "You are TestAI") {
		t.Errorf("Expected system prompt when no messages exist")
	}

	// Test with existing user message
	payload2 := []byte(`{"model":"test-model","messages":[{"role":"user","content":"Hello"}]}`)
	result2 := applySystemPromptRules(cfg, "test-model", "openai", payload2, "test-model")
	t.Logf("Result with user message: %s", string(result2))

	if !contains(string(result2), `"role":"system"`) {
		t.Errorf("Expected system role in messages")
	}
}

func TestSystemPromptIntegration_AliasMatching(t *testing.T) {
	cfg := &config.Config{
		Payload: config.PayloadConfig{
			SystemPrompts: []config.SystemPromptRule{
				{
					Models: []config.PayloadModelRule{
						{Name: "can", Protocol: "openai"},
					},
					Prompt: "You are Can, an AI assistant provided by Kodizm.",
					Mode:   "prepend",
				},
			},
		},
	}

	payload := []byte(`{"model":"kimi-k2.5","messages":[{"role":"user","content":"Who are you?"}]}`)

	resultNoAlias := applySystemPromptRules(cfg, "kimi-k2.5", "openai", payload, "")
	if contains(string(resultNoAlias), "You are Can") {
		t.Error("Should NOT match upstream model kimi-k2.5 against rule for 'can'")
	}

	resultWithAlias := applySystemPromptRules(cfg, "kimi-k2.5", "openai", payload, "can")
	if !contains(string(resultWithAlias), "You are Can") {
		t.Errorf("Should match alias 'can' via requestedModel, got: %s", string(resultWithAlias))
	}

	t.Logf("Alias matching works: %s", string(resultWithAlias))
}
