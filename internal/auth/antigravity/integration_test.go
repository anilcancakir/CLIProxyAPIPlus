package antigravity

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegrationQuotaPipeline(t *testing.T) {
	// 1. Mock Google API server returns 2 models with quota info
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := map[string]interface{}{
			"models": map[string]interface{}{
				"gemini-2.5-pro": map[string]interface{}{
					"displayName": "Gemini 2.5 Pro",
					"quotaInfo": map[string]interface{}{
						"remainingFraction": 0.75,
						"resetTime":         "2026-03-01T00:00:00Z",
					},
				},
				"gemini-2.5-flash": map[string]interface{}{
					"displayName": "Gemini 2.5 Flash",
					"quotaInfo": map[string]interface{}{
						"remainingFraction": 0.0,
						"resetTime":         "2026-03-01T00:00:00Z",
					},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// 2. Create checker pointed at mock server
	checker := NewAntigravityQuotaChecker(server.Client())
	checker.baseURL = server.URL // override base URL for test

	// 3. Fetch quota
	ctx := context.Background()
	quotaData, err := checker.FetchQuota(ctx, "fake-token", "test@example.com", "account-1")
	require.NoError(t, err)
	require.NotNil(t, quotaData)

	// 4. Assert 2 models with correct data
	assert.Len(t, quotaData.Models, 2)
	assert.False(t, quotaData.IsForbidden)

	// Find gemini-2.5-pro
	var proModel *ModelQuota
	for i := range quotaData.Models {
		if quotaData.Models[i].Name == "gemini-2.5-pro" {
			proModel = &quotaData.Models[i]
		}
	}
	require.NotNil(t, proModel)
	assert.InDelta(t, 0.75, proModel.RemainingFraction, 0.001)
	assert.InDelta(t, 75.0, proModel.RemainingPercent, 0.001)
	assert.False(t, proModel.ResetTime.IsZero())

	// 5. Verify cache contains the data
	cached := checker.GetCachedQuota("account-1")
	require.NotNil(t, cached)
	assert.Equal(t, len(quotaData.Models), len(cached.Models))
}

func TestIntegrationRateLimiterAndQuota(t *testing.T) {
	rl := NewAntigravityRateLimiter()
	checker := NewAntigravityQuotaChecker(&http.Client{})

	// Store quota directly (simulating FetchAntigravityModels extraction)
	model := "gemini-2.5-pro"
	checker.StoreQuota("account-1", &QuotaData{
		Models: []ModelQuota{
			{Name: model, RemainingFraction: 0.1, RemainingPercent: 10},
		},
		LastUpdated: time.Now(),
	})

	// Trigger quota exhaustion for that model
	modelPtr := model
	rl.ParseFromError("account-1", 429, nil, []byte(`{"error":{"message":"quota exhausted"}}`), &modelPtr, QuotaExhausted)

	// Model should be locked
	assert.True(t, rl.IsRateLimited("account-1", &modelPtr))

	// Other model should NOT be locked
	otherModel := "gemini-2.5-flash"
	assert.False(t, rl.IsRateLimited("account-1", &otherModel))

	// Quota data should still be accessible
	cached := checker.GetCachedQuota("account-1")
	require.NotNil(t, cached)
	assert.Len(t, cached.Models, 1)
	assert.Equal(t, model, cached.Models[0].Name)
}

func TestIntegrationStoreQuotaUpdatesCache(t *testing.T) {
	checker := NewAntigravityQuotaChecker(&http.Client{})

	// Store initial data
	checker.StoreQuota("acc-1", &QuotaData{
		Models:      []ModelQuota{{Name: "gemini-2.5-pro", RemainingFraction: 0.5}},
		LastUpdated: time.Now(),
	})

	// Store updated data
	checker.StoreQuota("acc-1", &QuotaData{
		Models:      []ModelQuota{{Name: "gemini-2.5-pro", RemainingFraction: 0.2}},
		LastUpdated: time.Now(),
	})

	cached := checker.GetCachedQuota("acc-1")
	require.NotNil(t, cached)
	assert.InDelta(t, 0.2, cached.Models[0].RemainingFraction, 0.001, "cache should have latest value")
}

func TestIntegrationConcurrent429(t *testing.T) {
	rl := NewAntigravityRateLimiterWithSteps(defaultBackoffSteps)
	model := "gemini-2.5-pro"

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rl.ParseFromError("acc-concurrent", 429, nil, []byte(`{}`), &model, QuotaExhausted)
		}()
	}
	wg.Wait()

	// Failure count CAN go up to 10 due to concurrency, but the backoff should be at most
	// the last step (7200s). The key is NO DATA RACE and backoff within bounds.
	wait := rl.GetRemainingWait("acc-concurrent", &model)
	assert.True(t, wait > 0, "should have a lockout duration")
	assert.True(t, wait <= 7200*time.Second+5*time.Second, "backoff should not exceed max step + buffer")
}

func TestIntegrationRateLimitExceededBlocksAll(t *testing.T) {
	rl := NewAntigravityRateLimiter()
	model1 := "gemini-2.5-pro"
	model2 := "gemini-2.5-flash"

	// RateLimitExceeded → account-level lock
	rl.ParseFromError("acc-1", 429, nil, []byte(`{}`), nil, RateLimitExceeded)

	assert.True(t, rl.IsRateLimited("acc-1", nil), "account level locked")
	assert.True(t, rl.IsRateLimited("acc-1", &model1), "model1 blocked by account lock")
	assert.True(t, rl.IsRateLimited("acc-1", &model2), "model2 blocked by account lock")

	// Different account should NOT be locked
	assert.False(t, rl.IsRateLimited("acc-2", nil))
}

func TestIntegrationServerErrorDoesNotAffect429(t *testing.T) {
	rl := NewAntigravityRateLimiter()
	model := "gemini-2.5-pro"

	// 3 server errors
	for i := 0; i < 3; i++ {
		rl.ParseFromError("acc-1", 503, nil, []byte(`{"error":{"message":"internal server error"}}`), nil, ServerError)
	}

	// Now a quota exhaustion — should STILL be tier 1 (60s)
	info := rl.ParseFromError("acc-1", 429, nil, []byte(`{}`), &model, QuotaExhausted)
	require.NotNil(t, info)
	assert.InDelta(t, 60.0, float64(info.RetryAfterSec), 5.0,
		"ServerErrors must not pollute 429 failure count")
}

func TestIntegrationConcurrentSafe(t *testing.T) {
	checker := NewAntigravityQuotaChecker(&http.Client{})
	rl := NewAntigravityRateLimiter()
	model := "gemini-2.5-pro"

	var wg sync.WaitGroup

	// 5 goroutines writing quota
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			checker.StoreQuota("acc-race", &QuotaData{
				Models:      []ModelQuota{{Name: model, RemainingFraction: float64(i) * 0.1}},
				LastUpdated: time.Now(),
			})
		}(i)
	}

	// 5 goroutines reading quota
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = checker.GetCachedQuota("acc-race")
		}()
	}

	// 5 goroutines writing rate limits
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rl.ParseFromError("acc-race", 429, nil, []byte(`{}`), &model, QuotaExhausted)
		}()
	}

	// 5 goroutines reading rate limits
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = rl.IsRateLimited("acc-race", &model)
			_ = rl.GetRemainingWait("acc-race", &model)
		}()
	}

	wg.Wait()
}
