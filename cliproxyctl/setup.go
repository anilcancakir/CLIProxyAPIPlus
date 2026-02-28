// Package main provides the cliproxyctl management CLI binary for CLIProxyAPI.
package main

import (
	"fmt"
	"strings"

	"github.com/router-for-me/CLIProxyAPI/v6/internal/config"
)

// SetupOptions controls interactive wizard behaviour.
type SetupOptions struct {
	// ConfigPath points to the active config file.
	ConfigPath string

	// Prompt provides custom prompt handling for tests.
	Prompt func(string) (string, error)
}

// DoSetupWizard runs a minimal first-run setup summary.
// In the origin repository, a full interactive wizard is not implemented; this
// function prints a diagnostic overview of the loaded configuration.
//
// Parameters:
//   - cfg: The application configuration.
//   - options: Setup wizard options including the config path.
func DoSetupWizard(cfg *config.Config, options *SetupOptions) {
	if cfg == nil {
		cfg = &config.Config{}
	}

	configPath := ""
	if options != nil {
		configPath = options.ConfigPath
	}

	fmt.Println("cliproxyctl setup summary")
	fmt.Printf("  config: %s\n", emptyOrDefault(configPath, "(default)"))
	fmt.Printf(
		"  providers configured: codex=%d, claude=%d, gemini=%d, kiro=%d, openai-compat=%d\n",
		len(cfg.CodexKey),
		len(cfg.ClaudeKey),
		len(cfg.GeminiKey),
		len(cfg.KiroKey),
		len(cfg.OpenAICompatibility),
	)
}

// emptyOrDefault returns fallback when value is empty after trimming.
func emptyOrDefault(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}

	return value
}
