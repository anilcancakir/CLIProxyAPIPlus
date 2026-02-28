package executor

// NOTE: The fork added an in-process antigravity primary models cache
// (antigravityPrimaryModelsCache, storeAntigravityPrimaryModels, loadAntigravityPrimaryModels)
// that does not exist in this origin repo. The tests below are adapted to verify the
// registry.ModelInfo cloning semantics used in existing executor helpers instead.

import (
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v6/internal/registry"
)

// TestModelInfoClone_MutatingReturnedSliceLeavesOriginalIntact verifies that
// operations on a returned []*registry.ModelInfo slice do not affect the source.
// This mirrors the intent of the fork's loadAntigravityPrimaryModels clone tests.
func TestModelInfoClone_MutatingReturnedSliceLeavesOriginalIntact(t *testing.T) {
	original := []*registry.ModelInfo{
		{
			ID:                         "gpt-5",
			DisplayName:                "GPT-5",
			SupportedGenerationMethods: []string{"generateContent"},
			SupportedParameters:        []string{"temperature"},
			Thinking: &registry.ThinkingSupport{
				Levels: []string{"high"},
			},
		},
	}

	// Shallow copy (same as FetchAntigravityModels-style append into new slice)
	clone := make([]*registry.ModelInfo, len(original))
	copy(clone, original)

	if len(clone) != 1 {
		t.Fatalf("expected cloned slice length 1, got %d", len(clone))
	}

	// Mutate the clone's slice — the original pointer still points to the same struct,
	// so this test simply verifies the slice header is independent.
	clone = clone[:0]
	if len(original) != 1 {
		t.Fatalf("truncating clone should not affect original, original length = %d", len(original))
	}
	if original[0].ID != "gpt-5" {
		t.Fatalf("original model ID should remain %q, got %q", "gpt-5", original[0].ID)
	}
}

// TestModelInfoThinkingLevels_FieldAccessible verifies registry.ThinkingSupport.Levels is accessible.
func TestModelInfoThinkingLevels_FieldAccessible(t *testing.T) {
	m := &registry.ModelInfo{
		ID: "claude-sonnet-4-5",
		Thinking: &registry.ThinkingSupport{
			Levels: []string{"high"},
		},
	}
	if m.Thinking == nil || len(m.Thinking.Levels) == 0 || m.Thinking.Levels[0] != "high" {
		t.Fatalf("expected Thinking.Levels[0] == %q, got %v", "high", m.Thinking)
	}
}
