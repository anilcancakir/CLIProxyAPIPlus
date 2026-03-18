package executor

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// isClaudeCodeClient Tests
// ---------------------------------------------------------------------------

func TestIsClaudeCodeClient_ClaudeCLIPrefix(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		ua       string
		expected bool
	}{
		{
			name:     "real Claude Code UA",
			ua:       "claude-cli/2.1.76 (external)",
			expected: true,
		},
		{
			name:     "older Claude Code UA",
			ua:       "claude-cli/2.0.0 (external, cli)",
			expected: true,
		},
		{
			name:     "bare prefix",
			ua:       "claude-cli",
			expected: true,
		},
		{
			name:     "curl UA",
			ua:       "curl/8.0",
			expected: false,
		},
		{
			name:     "OpenAI SDK",
			ua:       "OpenAI/Python 1.0.0",
			expected: false,
		},
		{
			name:     "empty",
			ua:       "",
			expected: false,
		},
		{
			name:     "claude without cli suffix",
			ua:       "claude-code/1.0",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isClaudeCodeClient(tc.ua); got != tc.expected {
				t.Fatalf("isClaudeCodeClient(%q) = %v, want %v", tc.ua, got, tc.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// shouldCloak Tests
// ---------------------------------------------------------------------------

func TestShouldCloak_AlwaysMode(t *testing.T) {
	t.Parallel()

	// "always" should cloak even for Claude Code clients.
	if got := shouldCloak("always", "claude-cli/2.1.76 (external)"); !got {
		t.Fatal("shouldCloak(always, claude-cli) = false, want true")
	}
}

func TestShouldCloak_NeverMode(t *testing.T) {
	t.Parallel()

	// "never" should not cloak even for non-Claude clients.
	if got := shouldCloak("never", "curl/8.0"); got {
		t.Fatal("shouldCloak(never, curl) = true, want false")
	}
}

func TestShouldCloak_AutoMode_ClaudeClient(t *testing.T) {
	t.Parallel()

	// "auto" with Claude Code client → don't cloak.
	if got := shouldCloak("auto", "claude-cli/2.1.76 (external)"); got {
		t.Fatal("shouldCloak(auto, claude-cli) = true, want false")
	}
}

func TestShouldCloak_AutoMode_NonClaudeClient(t *testing.T) {
	t.Parallel()

	// "auto" with non-Claude client → cloak.
	if got := shouldCloak("auto", "curl/8.0"); !got {
		t.Fatal("shouldCloak(auto, curl) = false, want true")
	}
}

func TestShouldCloak_EmptyMode_DefaultsToAuto(t *testing.T) {
	t.Parallel()

	// Empty mode defaults to auto behavior.
	if got := shouldCloak("", "curl/8.0"); !got {
		t.Fatal("shouldCloak('', curl) = false, want true")
	}
	if got := shouldCloak("", "claude-cli/2.1.76"); got {
		t.Fatal("shouldCloak('', claude-cli) = true, want false")
	}
}

// ---------------------------------------------------------------------------
// generateFakeUserID Tests
// ---------------------------------------------------------------------------

func TestGenerateFakeUserID_MatchesClaudeCodeFormat(t *testing.T) {
	t.Parallel()

	id := generateFakeUserID()
	if !isValidUserID(id) {
		t.Fatalf("generateFakeUserID() = %q, does not match Claude Code format", id)
	}
}

func TestGenerateFakeUserID_HasCorrectPrefix(t *testing.T) {
	t.Parallel()

	id := generateFakeUserID()
	if !strings.HasPrefix(id, "user_") {
		t.Fatalf("generateFakeUserID() = %q, missing 'user_' prefix", id)
	}
}

func TestGenerateFakeUserID_HasAccountAndSessionUUID(t *testing.T) {
	t.Parallel()

	id := generateFakeUserID()
	if !strings.Contains(id, "_account_") {
		t.Fatalf("generateFakeUserID() = %q, missing '_account_'", id)
	}
	if !strings.Contains(id, "_session_") {
		t.Fatalf("generateFakeUserID() = %q, missing '_session_'", id)
	}
}

func TestGenerateFakeUserID_TwoCallsProduceDifferentIDs(t *testing.T) {
	t.Parallel()

	id1 := generateFakeUserID()
	id2 := generateFakeUserID()
	if id1 == id2 {
		t.Fatal("two consecutive calls produced identical IDs")
	}
}

// ---------------------------------------------------------------------------
// isValidUserID Tests
// ---------------------------------------------------------------------------

func TestIsValidUserID_ValidFormat(t *testing.T) {
	t.Parallel()

	valid := "user_" + strings.Repeat("a", 64) + "_account_12345678-1234-1234-1234-123456789012_session_12345678-1234-1234-1234-123456789012"
	if !isValidUserID(valid) {
		t.Fatalf("isValidUserID(%q) = false, want true", valid)
	}
}

func TestIsValidUserID_InvalidFormats(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"no prefix", strings.Repeat("a", 64) + "_account_12345678-1234-1234-1234-123456789012_session_12345678-1234-1234-1234-123456789012"},
		{"short hex", "user_" + strings.Repeat("a", 32) + "_account_12345678-1234-1234-1234-123456789012_session_12345678-1234-1234-1234-123456789012"},
		{"no session", "user_" + strings.Repeat("a", 64) + "_account_12345678-1234-1234-1234-123456789012"},
		{"random string", "not_a_valid_user_id_at_all"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if isValidUserID(tc.input) {
				t.Fatalf("isValidUserID(%q) = true, want false", tc.input)
			}
		})
	}
}
