package executor

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
	"testing"
)

func makePayload(t *testing.T, userMsg string) []byte {
	t.Helper()
	payload := []byte(`{"messages":[{"role":"user","content":"` + userMsg + `"}]}`)
	return payload
}

func TestExtractChar(t *testing.T) {
	testCases := []struct {
		name     string
		text     string
		index    int
		expected string
	}{
		{
			name:     "returns in-bounds index 4",
			text:     "Hello world",
			index:    4,
			expected: "o",
		},
		{
			name:     "returns in-bounds index 7",
			text:     "Hello world",
			index:    7,
			expected: "o",
		},
		{
			name:     "returns undefined for out-of-bounds on normal text",
			text:     "Hello world",
			index:    20,
			expected: "undefined",
		},
		{
			name:     "returns first character",
			text:     "hi",
			index:    0,
			expected: "h",
		},
		{
			name:     "returns undefined for out-of-bounds on short text",
			text:     "hi",
			index:    4,
			expected: "undefined",
		},
		{
			name:     "returns undefined for empty string",
			text:     "",
			index:    0,
			expected: "undefined",
		},
		{
			name:     "returns exact boundary index 20 when len is 21",
			text:     "abcdefghijklmnopqrstu",
			index:    20,
			expected: "u",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			got := extractCharWithFallback(testCase.text, testCase.index)
			if got != testCase.expected {
				t.Fatalf("extractCharWithFallback(%q, %d) = %q, want %q", testCase.text, testCase.index, got, testCase.expected)
			}
		})
	}
}

func TestGetFirstUserMessage(t *testing.T) {
	testCases := []struct {
		name     string
		payload  []byte
		expected string
	}{
		{
			name:     "returns empty string when no messages",
			payload:  []byte(`{"model":"claude-3-5-sonnet"}`),
			expected: "",
		},
		{
			name:     "returns empty string when only assistant messages",
			payload:  []byte(`{"messages":[{"role":"assistant","content":"hello"}]}`),
			expected: "",
		},
		{
			name:     "returns user string content",
			payload:  []byte(`{"messages":[{"role":"user","content":"plain text"}]}`),
			expected: "plain text",
		},
		{
			name:     "returns user array text block",
			payload:  []byte(`{"messages":[{"role":"user","content":[{"type":"text","text":"block text"}]}]}`),
			expected: "block text",
		},
		{
			name: "returns first text when thinking block appears first",
			payload: []byte(
				`{"messages":[{"role":"user","content":[{"type":"thinking","thinking":"internal"},{"type":"text","text":"final text"}]}]}`,
			),
			expected: "final text",
		},
		{
			name: "returns empty string when user array contains only thinking blocks",
			payload: []byte(
				`{"messages":[{"role":"user","content":[{"type":"thinking","thinking":"a"},{"type":"redacted_thinking","thinking":"b"}]}]}`,
			),
			expected: "",
		},
		{
			name: "returns first user message text among multiple users",
			payload: []byte(
				`{"messages":[{"role":"user","content":"first"},{"role":"assistant","content":"mid"},{"role":"user","content":"second"}]}`,
			),
			expected: "first",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			got := getFirstUserMessageText(testCase.payload)
			if got != testCase.expected {
				t.Fatalf("getFirstUserMessageText(%s) = %q, want %q", string(testCase.payload), got, testCase.expected)
			}
		})
	}
}

func TestBillingHeader(t *testing.T) {
	testCases := []struct {
		name    string
		payload []byte
	}{
		{
			name:    "billing header format is valid",
			payload: makePayload(t, "Hello, this is a test message for billing"),
		},
	}

	pattern := regexp.MustCompile(`^x-anthropic-billing-header: cc_version=2\.1\.63\.([0-9a-f]{3}); cc_entrypoint=cli; cch=(.+);$`)

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			header := generateBillingHeader(testCase.payload)
			matches := pattern.FindStringSubmatch(header)
			if len(matches) != 3 {
				t.Fatalf("header does not match expected format: %q", header)
			}

			buildHash := matches[1]
			cch := matches[2]

			if len(buildHash) != 3 {
				t.Fatalf("buildHash length = %d, want 3", len(buildHash))
			}
			if cch != "00000" {
				t.Fatalf("cch = %q, want %q", cch, "00000")
			}
		})
	}
}

func TestGenerateBillingHeader(t *testing.T) {
	testCases := []struct {
		name              string
		payload           []byte
		expectedHashInput string
		assertDeterminism bool
	}{
		{
			name:              "same payload generates deterministic header",
			payload:           makePayload(t, "Hello, this is a test message for billing"),
			expectedHashInput: "59cf53e54c78ott2.1.42",
			assertDeterminism: true,
		},
		{
			name:              "empty messages uses undefined fallback values",
			payload:           []byte(`{"messages":[]}`),
			expectedHashInput: "59cf53e54c78undefinedundefinedundefined2.1.42",
		},
		{
			name:              "short first user message mixes real and undefined characters",
			payload:           makePayload(t, "hello"),
			expectedHashInput: "59cf53e54c78oundefinedundefined2.1.42",
		},
		{
			name:              "normal first user message uses real characters at all positions",
			payload:           makePayload(t, "abcdefghijklmnopqrstu"),
			expectedHashInput: "59cf53e54c78ehu2.1.42",
		},
	}

	pattern := regexp.MustCompile(`^x-anthropic-billing-header: cc_version=2\.1\.63\.([0-9a-f]{3}); cc_entrypoint=cli; cch=(00000);$`)

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			header := generateBillingHeader(testCase.payload)

			if !strings.HasPrefix(header, "x-anthropic-billing-header: cc_version=2.1.63.") {
				t.Fatalf("header prefix mismatch: %q", header)
			}

			matches := pattern.FindStringSubmatch(header)
			if len(matches) != 3 {
				t.Fatalf("header format mismatch: %q", header)
			}

			buildHash := matches[1]
			if len(buildHash) != 3 {
				t.Fatalf("buildHash length = %d, want 3", len(buildHash))
			}

			expectedHashBytes := sha256.Sum256([]byte(testCase.expectedHashInput))
			expectedBuildHash := hex.EncodeToString(expectedHashBytes[:])[:3]
			if buildHash != expectedBuildHash {
				t.Fatalf("buildHash = %q, want %q", buildHash, expectedBuildHash)
			}

			if testCase.assertDeterminism {
				header2 := generateBillingHeader(testCase.payload)
				if header != header2 {
					t.Fatalf("generateBillingHeader is not deterministic: %q != %q", header, header2)
				}
			}
		})
	}
}
