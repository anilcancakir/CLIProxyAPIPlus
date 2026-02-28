package antigravity

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRateLimitReasonString(t *testing.T) {
	tests := []struct {
		reason   RateLimitReason
		expected string
	}{
		{
			reason:   Unknown,
			expected: "UNKNOWN",
		},
		{
			reason:   QuotaExhausted,
			expected: "QUOTA_EXHAUSTED",
		},
		{
			reason:   RateLimitExceeded,
			expected: "RATE_LIMIT_EXCEEDED",
		},
		{
			reason:   ModelCapacityExhausted,
			expected: "MODEL_CAPACITY_EXHAUSTED",
		},
		{
			reason:   ServerError,
			expected: "SERVER_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if tt.reason.String() != tt.expected {
				t.Errorf(
					"expected %s, got %s",
					tt.expected,
					tt.reason.String(),
				)
			}
		})
	}
}

func TestRateLimitInfoIsModelLevel(t *testing.T) {
	modelName := "gemini-2.5-pro"
	tests := []struct {
		name     string
		model    *string
		expected bool
	}{
		{
			name:     "account level",
			model:    nil,
			expected: false,
		},
		{
			name:     "model level",
			model:    &modelName,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := RateLimitInfo{
				Model: tt.model,
			}
			if info.IsModelLevel() != tt.expected {
				t.Errorf(
					"expected IsModelLevel() to be %v",
					tt.expected,
				)
			}
		})
	}
}

func TestRateLimiterInterface(t *testing.T) {
	var _ RateLimiter = (RateLimiter)(nil)
}


func TestQuotaExhaustedBackoff(t *testing.T) {
	rl := NewAntigravityRateLimiterWithSteps([]time.Duration{
		60 * time.Second, 300 * time.Second, 1800 * time.Second, 7200 * time.Second,
	})
	model := "gemini-2.5-pro"
	// First 429 → ~60s
	info1 := rl.ParseFromError("acc1", 429, nil, []byte(`{}`), &model, QuotaExhausted)
	assert.InDelta(t, 60.0, float64(info1.RetryAfterSec), 2.0)
	rl.CleanupExpired() // clear for next test
	// To simulate second failure, call again without cleanup
	rl2 := NewAntigravityRateLimiterWithSteps(defaultBackoffSteps)
	rl2.ParseFromError("acc1", 429, nil, []byte(`{}`), &model, QuotaExhausted) // failure 1
	info2 := rl2.ParseFromError("acc1", 429, nil, []byte(`{}`), &model, QuotaExhausted) // failure 2
	assert.InDelta(t, 300.0, float64(info2.RetryAfterSec), 2.0)
}

func TestServerErrorIsolation(t *testing.T) {
	rl := NewAntigravityRateLimiter()
	// 5x ServerError → 8s each, failure count stays 0
	for i := 0; i < 5; i++ {
		info := rl.ParseFromError("acc1", 503, nil, []byte(`{}`), nil, ServerError)
		assert.InDelta(t, 8.0, float64(info.RetryAfterSec), 2.0)
	}
	// Now QuotaExhausted → should be tier 1 (60s), not tier 4 (7200s)
	model := "gemini-2.5-pro"
	info := rl.ParseFromError("acc1", 429, nil, []byte(`{}`), &model, QuotaExhausted)
	assert.InDelta(t, 60.0, float64(info.RetryAfterSec), 2.0, "ServerError must not pollute QuotaExhausted failure count")
}

func TestModelIsolation(t *testing.T) {
	rl := NewAntigravityRateLimiter()
	model1 := "gemini-2.5-pro"
	model2 := "gemini-2.5-flash"
	// Lock model1
	rl.ParseFromError("acc1", 429, nil, []byte(`{}`), &model1, QuotaExhausted)
	assert.True(t, rl.IsRateLimited("acc1", &model1), "model1 should be locked")
	assert.False(t, rl.IsRateLimited("acc1", &model2), "model2 should NOT be locked")
	assert.False(t, rl.IsRateLimited("acc1", nil), "account-level should NOT be locked")
}

func TestRateLimitExceededAccountWide(t *testing.T) {
	rl := NewAntigravityRateLimiter()
	anyModel := "any-model"
	rl.ParseFromError("acc1", 429, nil, []byte(`{}`), nil, RateLimitExceeded)
	assert.True(t, rl.IsRateLimited("acc1", nil), "account level should be locked")
	assert.True(t, rl.IsRateLimited("acc1", &anyModel), "model check should see account-level lock")
}

func TestRateLimiterConcurrent(t *testing.T) {
	rl := NewAntigravityRateLimiter()
	model := "gemini-2.5-pro"
	done := make(chan struct{})
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			rl.ParseFromError("acc1", 429, nil, []byte(`{}`), &model, QuotaExhausted)
			rl.IsRateLimited("acc1", &model)
			rl.GetRemainingWait("acc1", &model)
		}()
	}
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestMarkSuccessClearsFailureCount(t *testing.T) {
	rl := NewAntigravityRateLimiter()
	model := "gemini-2.5-pro"
	// Two failures → tier 2 (300s)
	rl.ParseFromError("acc1", 429, nil, []byte(`{}`), &model, QuotaExhausted)
	rl.ParseFromError("acc1", 429, nil, []byte(`{}`), &model, QuotaExhausted)
	// Mark success clears failure count
	rl.MarkSuccess("acc1")
	// Next failure should be tier 1 again (60s)
	info := rl.ParseFromError("acc1", 429, nil, []byte(`{}`), &model, QuotaExhausted)
	assert.InDelta(t, 60.0, float64(info.RetryAfterSec), 2.0, "After MarkSuccess, backoff should reset to tier 1")
}