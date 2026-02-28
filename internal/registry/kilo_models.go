// Package registry provides model definitions for various AI service providers.
package registry

// GetKiloModels returns the static Kilo model definitions.
// The first entry (kilo/auto) is always prepended to any dynamically fetched models.
func GetKiloModels() []*ModelInfo {
	return []*ModelInfo{
		// --- Base Models ---
		{
			ID:                  "kilo/auto",
			Object:              "model",
			Created:             1732752000,
			OwnedBy:             "kilo",
			Type:                "kilo",
			DisplayName:         "Kilo Auto",
			Description:         "Automatic model selection by Kilo",
			ContextLength:       200000,
			MaxCompletionTokens: 64000,
			Thinking:            &ThinkingSupport{Min: 1024, Max: 32000, ZeroAllowed: true, DynamicAllowed: true},
		},
	}
}
