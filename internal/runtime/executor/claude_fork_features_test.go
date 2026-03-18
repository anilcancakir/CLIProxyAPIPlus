package executor

import (
	"net/http"
	"strings"
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v6/internal/config"
	cliproxyauth "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/auth"
	"github.com/tidwall/gjson"
)

// ---------------------------------------------------------------------------
// isClaudeOAuthToken Tests
// ---------------------------------------------------------------------------

func TestIsClaudeOAuthToken_OAuthToken(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		apiKey   string
		expected bool
	}{
		{
			name:     "standard OAuth token",
			apiKey:   "sk-ant-oat-abc123",
			expected: true,
		},
		{
			name:     "OAuth token in middle",
			apiKey:   "prefix-sk-ant-oat-suffix",
			expected: true,
		},
		{
			name:     "direct API key",
			apiKey:   "sk-ant-api01-abc123",
			expected: false,
		},
		{
			name:     "empty",
			apiKey:   "",
			expected: false,
		},
		{
			name:     "GitHub Copilot token",
			apiKey:   "ghu_abc123",
			expected: false,
		},
		{
			name:     "random string",
			apiKey:   "some-random-key",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isClaudeOAuthToken(tc.apiKey); got != tc.expected {
				t.Fatalf("isClaudeOAuthToken(%q) = %v, want %v", tc.apiKey, got, tc.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// claudeCreds Tests
// ---------------------------------------------------------------------------

func TestClaudeCreds_NilAuth(t *testing.T) {
	t.Parallel()

	apiKey, baseURL := claudeCreds(nil)
	if apiKey != "" || baseURL != "" {
		t.Fatalf("claudeCreds(nil) = (%q, %q), want (\"\", \"\")", apiKey, baseURL)
	}
}

func TestClaudeCreds_FromAttributes(t *testing.T) {
	t.Parallel()

	auth := &cliproxyauth.Auth{
		Attributes: map[string]string{
			"api_key":  "sk-ant-api01-test",
			"base_url": "https://custom.api.com",
		},
	}

	apiKey, baseURL := claudeCreds(auth)
	if apiKey != "sk-ant-api01-test" {
		t.Fatalf("apiKey = %q, want %q", apiKey, "sk-ant-api01-test")
	}
	if baseURL != "https://custom.api.com" {
		t.Fatalf("baseURL = %q, want %q", baseURL, "https://custom.api.com")
	}
}

func TestClaudeCreds_FallbackToMetadata(t *testing.T) {
	t.Parallel()

	auth := &cliproxyauth.Auth{
		Attributes: map[string]string{},
		Metadata: map[string]interface{}{
			"access_token": "sk-ant-oat-from-metadata",
		},
	}

	apiKey, _ := claudeCreds(auth)
	if apiKey != "sk-ant-oat-from-metadata" {
		t.Fatalf("apiKey = %q, want %q", apiKey, "sk-ant-oat-from-metadata")
	}
}

func TestClaudeCreds_AttributesTakePrecedence(t *testing.T) {
	t.Parallel()

	auth := &cliproxyauth.Auth{
		Attributes: map[string]string{
			"api_key": "from-attributes",
		},
		Metadata: map[string]interface{}{
			"access_token": "from-metadata",
		},
	}

	apiKey, _ := claudeCreds(auth)
	if apiKey != "from-attributes" {
		t.Fatalf("apiKey = %q, want %q (attributes should take precedence)", apiKey, "from-attributes")
	}
}

// ---------------------------------------------------------------------------
// stripThinkingBlocks Tests
// ---------------------------------------------------------------------------

func TestStripThinkingBlocks_RemovesThinking(t *testing.T) {
	t.Parallel()

	payload := []byte(`{"messages":[{"role":"assistant","content":[{"type":"thinking","thinking":"internal thought"},{"type":"text","text":"visible text"}]}]}`)
	result := stripThinkingBlocks(payload)

	if gjson.GetBytes(result, "messages.0.content.#").Int() != 1 {
		t.Fatalf("expected 1 content block, got %d", gjson.GetBytes(result, "messages.0.content.#").Int())
	}
	if gjson.GetBytes(result, "messages.0.content.0.type").String() != "text" {
		t.Fatalf("remaining block type = %q, want 'text'", gjson.GetBytes(result, "messages.0.content.0.type").String())
	}
}

func TestStripThinkingBlocks_RemovesRedactedThinking(t *testing.T) {
	t.Parallel()

	payload := []byte(`{"messages":[{"role":"assistant","content":[{"type":"redacted_thinking","data":"xxx"},{"type":"text","text":"visible"}]}]}`)
	result := stripThinkingBlocks(payload)

	if gjson.GetBytes(result, "messages.0.content.#").Int() != 1 {
		t.Fatalf("expected 1 content block after stripping redacted_thinking")
	}
}

func TestStripThinkingBlocks_RemovesEmptyAssistantMessage(t *testing.T) {
	t.Parallel()

	// If all content blocks are thinking, the entire message should be removed.
	payload := []byte(`{"messages":[{"role":"user","content":"hello"},{"role":"assistant","content":[{"type":"thinking","thinking":"only thinking"}]},{"role":"user","content":"followup"}]}`)
	result := stripThinkingBlocks(payload)

	msgCount := gjson.GetBytes(result, "messages.#").Int()
	if msgCount != 2 {
		t.Fatalf("message count = %d, want 2 (user+user, assistant removed)", msgCount)
	}
	// Both remaining messages should be user role.
	for i := int64(0); i < msgCount; i++ {
		if role := gjson.GetBytes(result, "messages."+string(rune('0'+i))+".role").String(); role != "user" {
			t.Fatalf("messages[%d].role = %q, want 'user'", i, role)
		}
	}
}

func TestStripThinkingBlocks_NoOpWhenNoThinking(t *testing.T) {
	t.Parallel()

	payload := []byte(`{"messages":[{"role":"assistant","content":[{"type":"text","text":"no thinking here"}]}]}`)
	result := stripThinkingBlocks(payload)

	if string(result) != string(payload) {
		t.Fatalf("stripThinkingBlocks modified payload with no thinking blocks")
	}
}

func TestStripThinkingBlocks_NoOpForUserMessages(t *testing.T) {
	t.Parallel()

	payload := []byte(`{"messages":[{"role":"user","content":"just text"}]}`)
	result := stripThinkingBlocks(payload)

	if string(result) != string(payload) {
		t.Fatalf("stripThinkingBlocks should not modify user messages")
	}
}

func TestStripThinkingBlocks_HandlesStringContent(t *testing.T) {
	t.Parallel()

	// String content (not array) should be left untouched.
	payload := []byte(`{"messages":[{"role":"assistant","content":"plain string"}]}`)
	result := stripThinkingBlocks(payload)

	if gjson.GetBytes(result, "messages.0.content").String() != "plain string" {
		t.Fatal("string content was modified")
	}
}

// ---------------------------------------------------------------------------
// applyClaudeHeaders Tests
// ---------------------------------------------------------------------------

func TestApplyClaudeHeaders_DefaultUserAgent(t *testing.T) {
	t.Parallel()

	req, _ := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", nil)
	auth := &cliproxyauth.Auth{}
	applyClaudeHeaders(req, auth, "sk-test", false, nil, nil)

	ua := req.Header.Get("User-Agent")
	if ua != "claude-cli/2.1.76 (external)" {
		t.Fatalf("User-Agent = %q, want %q", ua, "claude-cli/2.1.76 (external)")
	}
}

func TestApplyClaudeHeaders_XAppHeader(t *testing.T) {
	t.Parallel()

	req, _ := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", nil)
	applyClaudeHeaders(req, nil, "sk-test", false, nil, nil)

	if got := req.Header.Get("X-App"); got != "cli" {
		t.Fatalf("X-App = %q, want %q", got, "cli")
	}
}

func TestApplyClaudeHeaders_StainlessHeaders(t *testing.T) {
	t.Parallel()

	req, _ := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", nil)
	applyClaudeHeaders(req, nil, "sk-test", false, nil, nil)

	expected := map[string]string{
		"X-Stainless-Package-Version": "0.74.0",
		"X-Stainless-Runtime":         "node",
		"X-Stainless-Lang":            "js",
		"X-Stainless-Runtime-Version": "v24.3.0",
		"X-Stainless-Retry-Count":     "0",
	}

	for header, want := range expected {
		if got := req.Header.Get(header); got != want {
			t.Errorf("%s = %q, want %q", header, got, want)
		}
	}
}

func TestApplyClaudeHeaders_BetaHeaders_Default(t *testing.T) {
	t.Parallel()

	req, _ := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", nil)
	applyClaudeHeaders(req, nil, "sk-test", false, nil, nil)

	beta := req.Header.Get("Anthropic-Beta")
	essentialBetas := []string{
		"claude-code-20250219",
		"oauth-2025-04-20",
		"interleaved-thinking-2025-05-14",
		"context-management-2025-06-27",
		"prompt-caching-scope-2026-01-05",
	}

	for _, b := range essentialBetas {
		if !strings.Contains(beta, b) {
			t.Errorf("Anthropic-Beta missing %q, got %q", b, beta)
		}
	}
}

func TestApplyClaudeHeaders_StreamingAcceptEncoding(t *testing.T) {
	t.Parallel()

	req, _ := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", nil)
	applyClaudeHeaders(req, nil, "sk-test", true, nil, nil)

	if got := req.Header.Get("Accept-Encoding"); got != "identity" {
		t.Fatalf("streaming Accept-Encoding = %q, want 'identity'", got)
	}
}

func TestApplyClaudeHeaders_NonStreamingAcceptEncoding(t *testing.T) {
	t.Parallel()

	req, _ := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", nil)
	applyClaudeHeaders(req, nil, "sk-test", false, nil, nil)

	got := req.Header.Get("Accept-Encoding")
	if !strings.Contains(got, "gzip") {
		t.Fatalf("non-streaming Accept-Encoding = %q, want to contain 'gzip'", got)
	}
}

func TestApplyClaudeHeaders_OAuthTokenUsesBearer(t *testing.T) {
	t.Parallel()

	req, _ := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", nil)
	auth := &cliproxyauth.Auth{
		Metadata: map[string]interface{}{"access_token": "sk-ant-oat-test"},
	}
	applyClaudeHeaders(req, auth, "sk-ant-oat-test", false, nil, nil)

	if got := req.Header.Get("Authorization"); got != "Bearer sk-ant-oat-test" {
		t.Fatalf("Authorization = %q, want 'Bearer sk-ant-oat-test'", got)
	}
}

func TestApplyClaudeHeaders_DirectAPIKeyUsesXApiKey(t *testing.T) {
	t.Parallel()

	req, _ := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", nil)
	auth := &cliproxyauth.Auth{
		Attributes: map[string]string{"api_key": "sk-ant-api01-direct"},
	}
	applyClaudeHeaders(req, auth, "sk-ant-api01-direct", false, nil, nil)

	if got := req.Header.Get("x-api-key"); got != "sk-ant-api01-direct" {
		t.Fatalf("x-api-key = %q, want 'sk-ant-api01-direct'", got)
	}
	// Authorization should not be set when using x-api-key on anthropic.com.
	if got := req.Header.Get("Authorization"); got != "" {
		t.Fatalf("Authorization = %q, want empty when using x-api-key", got)
	}
}

func TestApplyClaudeHeaders_ConfigOverridesUserAgent(t *testing.T) {
	t.Parallel()

	req, _ := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", nil)
	cfg := &config.Config{
		ClaudeHeaderDefaults: config.ClaudeHeaderDefaults{
			UserAgent: "custom-agent/1.0",
		},
	}
	applyClaudeHeaders(req, nil, "sk-test", false, nil, cfg)

	if got := req.Header.Get("User-Agent"); got != "custom-agent/1.0" {
		t.Fatalf("User-Agent = %q, want 'custom-agent/1.0'", got)
	}
}

func TestApplyClaudeHeaders_ExtraBetasAppended(t *testing.T) {
	t.Parallel()

	req, _ := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", nil)
	extraBetas := []string{"custom-beta-2025-01-01"}
	applyClaudeHeaders(req, nil, "sk-test", false, extraBetas, nil)

	beta := req.Header.Get("Anthropic-Beta")
	if !strings.Contains(beta, "custom-beta-2025-01-01") {
		t.Fatalf("Anthropic-Beta missing extra beta, got %q", beta)
	}
}

// ---------------------------------------------------------------------------
// flattenAssistantContent Tests (Copilot executor helper)
// ---------------------------------------------------------------------------

func TestFlattenAssistantContent_JoinsTextBlocks(t *testing.T) {
	t.Parallel()

	body := []byte(`{"messages":[{"role":"assistant","content":[{"type":"text","text":"Hello "},{"type":"text","text":"World"}]}]}`)
	result := flattenAssistantContent(body)

	content := gjson.GetBytes(result, "messages.0.content").String()
	if content != "Hello World" {
		t.Fatalf("flattened content = %q, want 'Hello World'", content)
	}
}

func TestFlattenAssistantContent_SkipsNonTextBlocks(t *testing.T) {
	t.Parallel()

	// Content with tool_use should NOT be flattened.
	body := []byte(`{"messages":[{"role":"assistant","content":[{"type":"text","text":"hello"},{"type":"tool_use","id":"t1","name":"foo","input":{}}]}]}`)
	result := flattenAssistantContent(body)

	// Should be left as array (not flattened).
	if !gjson.GetBytes(result, "messages.0.content").IsArray() {
		t.Fatal("content with non-text blocks should not be flattened")
	}
}

func TestFlattenAssistantContent_IgnoresUserMessages(t *testing.T) {
	t.Parallel()

	body := []byte(`{"messages":[{"role":"user","content":[{"type":"text","text":"user text"}]}]}`)
	result := flattenAssistantContent(body)

	// User messages should not be touched.
	if !gjson.GetBytes(result, "messages.0.content").IsArray() {
		t.Fatal("user message content should not be flattened")
	}
}

func TestFlattenAssistantContent_StringContentUntouched(t *testing.T) {
	t.Parallel()

	body := []byte(`{"messages":[{"role":"assistant","content":"already a string"}]}`)
	result := flattenAssistantContent(body)

	if gjson.GetBytes(result, "messages.0.content").String() != "already a string" {
		t.Fatal("string content was modified")
	}
}

func TestFlattenAssistantContent_EmptyMessages(t *testing.T) {
	t.Parallel()

	body := []byte(`{"messages":[]}`)
	result := flattenAssistantContent(body)

	if string(result) != string(body) {
		t.Fatal("empty messages should be returned unchanged")
	}
}
