package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractAccessToken(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		metadata map[string]any
		expected string
	}{
		{
			"antigravity top-level access_token",
			map[string]any{"access_token": "tok-abc"},
			"tok-abc",
		},
		{
			"gemini nested token.access_token",
			map[string]any{
				"token": map[string]any{"access_token": "tok-nested"},
			},
			"tok-nested",
		},
		{
			"top-level takes precedence over nested",
			map[string]any{
				"access_token": "tok-top",
				"token":        map[string]any{"access_token": "tok-nested"},
			},
			"tok-top",
		},
		{
			"empty metadata",
			map[string]any{},
			"",
		},
		{
			"whitespace-only access_token",
			map[string]any{"access_token": "   "},
			"",
		},
		{
			"wrong type access_token",
			map[string]any{"access_token": 12345},
			"",
		},
		{
			"token is not a map",
			map[string]any{"token": "not-a-map"},
			"",
		},
		{
			"nested whitespace-only",
			map[string]any{
				"token": map[string]any{"access_token": "  "},
			},
			"",
		},
		{
			"fallback to nested when top-level empty",
			map[string]any{
				"access_token": "",
				"token":        map[string]any{"access_token": "tok-fallback"},
			},
			"tok-fallback",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := extractAccessToken(tt.metadata)
			if got != tt.expected {
				t.Errorf("extractAccessToken() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestFilestore_ReadAuthFile_Priority(t *testing.T) {
	tempDir := t.TempDir()
	store := NewFileTokenStore()
	store.SetBaseDir(tempDir)

	tests := []struct {
		name             string
		metadata         map[string]any
		expectedPriority string
		priorityPresent  bool
	}{
		{
			name: "priority as integer 10",
			metadata: map[string]any{
				"type":     "copilot",
				"priority": 10,
			},
			expectedPriority: "10",
			priorityPresent:  true,
		},
		{
			name: "priority as float 10.0",
			metadata: map[string]any{
				"type":     "copilot",
				"priority": 10.0,
			},
			expectedPriority: "10",
			priorityPresent:  true,
		},
		{
			name: "priority as string 5",
			metadata: map[string]any{
				"type":     "copilot",
				"priority": "5",
			},
			expectedPriority: "5",
			priorityPresent:  true,
		},
		{
			name: "priority as explicit zero",
			metadata: map[string]any{
				"type":     "copilot",
				"priority": 0,
			},
			expectedPriority: "0",
			priorityPresent:  true,
		},
		{
			name: "priority as negative value",
			metadata: map[string]any{
				"type":     "copilot",
				"priority": -3,
			},
			expectedPriority: "-3",
			priorityPresent:  true,
		},
		{
			name: "missing priority key",
			metadata: map[string]any{
				"type": "copilot",
			},
			expectedPriority: "",
			priorityPresent:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filename := filepath.Join(tempDir, tt.name+".json")
			data, _ := json.Marshal(tt.metadata)
			_ = os.WriteFile(filename, data, 0644)

			auth, err := store.readAuthFile(filename, tempDir)
			if err != nil {
				t.Fatalf("failed to read auth file: %v", err)
			}

			priority, ok := auth.Attributes["priority"]
			if tt.priorityPresent {
				if !ok {
					t.Errorf("expected priority attribute to be present")
				}
				if priority != tt.expectedPriority {
					t.Errorf("expected priority %q, got %q", tt.expectedPriority, priority)
				}
			} else {
				if ok {
					t.Errorf("expected priority attribute to be absent, got %q", priority)
				}
			}
		})
	}
}
