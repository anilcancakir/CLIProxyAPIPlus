package antigravity

import (
	"testing"
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
