package usage

import (
	"testing"
	"time"

	coreusage "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/usage"
)

// TestPrune_RemovesStaleDetails verifies that records older than the cutoff are
// dropped and all aggregate counters are rebuilt from the surviving records only.
func TestPrune_RemovesStaleDetails(t *testing.T) {
	t.Parallel()

	s := NewRequestStatistics()
	now := time.Now()
	old := now.AddDate(0, 0, -8) // 8 days ago — outside 7-day window
	recent := now.AddDate(0, 0, -3)

	ingestAt := func(ts time.Time, tokens int64, failed bool) {
		s.Record(nil, coreusage.Record{
			APIKey:      "test-api",
			Model:       "claude-3",
			RequestedAt: ts,
			Failed:      failed,
			Detail: coreusage.Detail{
				TotalTokens: tokens,
			},
		})
	}

	ingestAt(old, 1000, false)    // should be pruned
	ingestAt(old, 500, true)      // should be pruned
	ingestAt(recent, 200, false)  // should survive
	ingestAt(recent, 300, false)  // should survive

	cutoff := now.AddDate(0, 0, -7)
	s.Prune(cutoff)

	snap := s.Snapshot()

	if snap.TotalRequests != 2 {
		t.Errorf("TotalRequests = %d, want 2", snap.TotalRequests)
	}

	if snap.SuccessCount != 2 {
		t.Errorf("SuccessCount = %d, want 2", snap.SuccessCount)
	}

	if snap.FailureCount != 0 {
		t.Errorf("FailureCount = %d, want 0", snap.FailureCount)
	}

	if snap.TotalTokens != 500 {
		t.Errorf("TotalTokens = %d, want 500", snap.TotalTokens)
	}

	apiSnap, ok := snap.APIs["test-api"]
	if !ok {
		t.Fatal("api 'test-api' missing from snapshot after prune")
	}

	modelSnap, ok := apiSnap.Models["claude-3"]
	if !ok {
		t.Fatal("model 'claude-3' missing from snapshot after prune")
	}

	if len(modelSnap.Details) != 2 {
		t.Errorf("Details count = %d, want 2", len(modelSnap.Details))
	}
}

// TestPrune_RemovesEmptyAPIBuckets verifies that API keys with no surviving records
// are cleaned up entirely from the statistics store.
func TestPrune_RemovesEmptyAPIBuckets(t *testing.T) {
	t.Parallel()

	s := NewRequestStatistics()
	old := time.Now().AddDate(0, 0, -10)

	s.Record(nil, coreusage.Record{
		APIKey:      "stale-api",
		Model:       "gpt-4",
		RequestedAt: old,
		Detail:      coreusage.Detail{TotalTokens: 100},
	})

	s.Prune(time.Now().AddDate(0, 0, -7))

	snap := s.Snapshot()

	if snap.TotalRequests != 0 {
		t.Errorf("TotalRequests = %d, want 0 after full prune", snap.TotalRequests)
	}

	if _, exists := snap.APIs["stale-api"]; exists {
		t.Error("stale-api should be removed from snapshot after full prune")
	}
}

// TestPrune_PreservesAllWhenNothingIsStale verifies that no data is lost when all
// records fall within the retention window.
func TestPrune_PreservesAllWhenNothingIsStale(t *testing.T) {
	t.Parallel()

	s := NewRequestStatistics()
	recent := time.Now().AddDate(0, 0, -1)

	for i := 0; i < 5; i++ {
		s.Record(nil, coreusage.Record{
			APIKey:      "api",
			Model:       "model",
			RequestedAt: recent,
			Detail:      coreusage.Detail{TotalTokens: 100},
		})
	}

	s.Prune(time.Now().AddDate(0, 0, -7))

	snap := s.Snapshot()

	if snap.TotalRequests != 5 {
		t.Errorf("TotalRequests = %d, want 5 — prune should not touch recent records", snap.TotalRequests)
	}
}
