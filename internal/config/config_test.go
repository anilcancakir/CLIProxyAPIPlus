package config

import (
	"reflect"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestConfig_OAuthProviderPriority_ParsesFromYAML(t *testing.T) {
	yamlData := `
oauth-provider-priority:
  claude: 10
  antigravity: 5
`
	var cfg Config
	err := yaml.Unmarshal([]byte(yamlData), &cfg)
	if err != nil {
		t.Fatalf("failed to unmarshal yaml: %v", err)
	}

	expected := map[string]int{
		"claude":      10,
		"antigravity": 5,
	}

	if !reflect.DeepEqual(cfg.OAuthProviderPriority, expected) {
		t.Errorf("expected OAuthProviderPriority %v, got %v", expected, cfg.OAuthProviderPriority)
	}
}

func TestConfig_OAuthProviderPriority_Missing(t *testing.T) {
	yamlData := `
port: 8080
`
	var cfg Config
	err := yaml.Unmarshal([]byte(yamlData), &cfg)
	if err != nil {
		t.Fatalf("failed to unmarshal yaml: %v", err)
	}

	if cfg.OAuthProviderPriority != nil {
		t.Errorf("expected OAuthProviderPriority to be nil, got %v", cfg.OAuthProviderPriority)
	}

	// Verify it returns 0 for missing keys without panic
	if val := cfg.OAuthProviderPriority["claude"]; val != 0 {
		t.Errorf("expected priority 0 for missing key, got %d", val)
	}
}

func TestConfig_OAuthProviderPriority_ZeroValue(t *testing.T) {
	yamlData := `
oauth-provider-priority:
  claude: 0
`
	var cfg Config
	err := yaml.Unmarshal([]byte(yamlData), &cfg)
	if err != nil {
		t.Fatalf("failed to unmarshal yaml: %v", err)
	}

	if val, ok := cfg.OAuthProviderPriority["claude"]; !ok || val != 0 {
		t.Errorf("expected priority 0 for claude, got %d (ok: %v)", val, ok)
	}
}

func TestConfig_QuotaExceeded_ClaudeQuotaThresholds_ParsesFromYAML(
	t *testing.T,
) {
	yamlData := `
quota-exceeded:
  switch-project: true
  switch-preview-model: false
  fallback-models:
    "claude-opus-4-5-20251101": "claude-sonnet-4-5-20250929"
  claude-quota-threshold:
    "claude-opus-4-5-20251101": 80.5
    "claude-sonnet-4-5-20250929": 95.0
`
	var cfg Config
	err := yaml.Unmarshal([]byte(yamlData), &cfg)
	if err != nil {
		t.Fatalf("failed to unmarshal yaml: %v", err)
	}

	expected := map[string]float64{
		"claude-opus-4-5-20251101":    80.5,
		"claude-sonnet-4-5-20250929": 95.0,
	}

	if !reflect.DeepEqual(cfg.QuotaExceeded.ClaudeQuotaThresholds, expected) {
		t.Errorf(
			"expected ClaudeQuotaThresholds %v, got %v",
			expected,
			cfg.QuotaExceeded.ClaudeQuotaThresholds,
		)
	}
}

func TestConfig_QuotaExceeded_ClaudeQuotaThresholds_Missing(
	t *testing.T,
) {
	yamlData := `
port: 8080
`
	var cfg Config
	err := yaml.Unmarshal([]byte(yamlData), &cfg)
	if err != nil {
		t.Fatalf("failed to unmarshal yaml: %v", err)
	}

	if cfg.QuotaExceeded.ClaudeQuotaThresholds != nil {
		t.Errorf(
			"expected ClaudeQuotaThresholds to be nil, got %v",
			cfg.QuotaExceeded.ClaudeQuotaThresholds,
		)
	}
}

func TestConfig_QuotaExceeded_ClaudeQuotaThresholds_EmptySection(
	t *testing.T,
) {
	yamlData := `
quota-exceeded:
  switch-project: true
`
	var cfg Config
	err := yaml.Unmarshal([]byte(yamlData), &cfg)
	if err != nil {
		t.Fatalf("failed to unmarshal yaml: %v", err)
	}

	if cfg.QuotaExceeded.ClaudeQuotaThresholds != nil {
		t.Errorf(
			"expected ClaudeQuotaThresholds to be nil when not specified, got %v",
			cfg.QuotaExceeded.ClaudeQuotaThresholds,
		)
	}
}
