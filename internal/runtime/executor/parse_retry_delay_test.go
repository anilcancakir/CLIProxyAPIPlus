package executor

import (
	"testing"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v6/internal/auth/antigravity"
	"github.com/stretchr/testify/assert"
)

func TestParseRetryDelayRetryInfo(t *testing.T) {
	body := []byte(`{
		"error": {
			"details": [
				{
					"@type": "type.googleapis.com/google.rpc.RetryInfo",
					"retryDelay": "0.847655010s"
				}
			]
		}
	}`)
	delay, err := parseRetryDelay(body)
	assert.NoError(t, err)
	assert.NotNil(t, delay)
	assert.Equal(t, 2*time.Second, *delay) // rounded up to 2s minimum
}

func TestParseRetryDelayQuotaReset(t *testing.T) {
	body := []byte(`{
		"error": {
			"details": [
				{
					"@type": "type.googleapis.com/google.rpc.ErrorInfo",
					"metadata": {
						"quotaResetDelay": "5s"
					}
				}
			]
		}
	}`)
	delay, err := parseRetryDelay(body)
	assert.NoError(t, err)
	assert.NotNil(t, delay)
	assert.Equal(t, 5*time.Second, *delay)
}

func TestParseRetryDelayAfterSeconds(t *testing.T) {
	body := []byte(`{"error": {"message": "Your quota will reset after 42s."}}`)
	delay, err := parseRetryDelay(body)
	assert.NoError(t, err)
	assert.NotNil(t, delay)
	assert.Equal(t, 42*time.Second, *delay)
}

func TestParseRetryDelayMinutesSeconds(t *testing.T) {
	body := []byte(`{"error": {"message": "Try again in 2m 30s"}}`)
	delay, err := parseRetryDelay(body)
	assert.NoError(t, err)
	assert.NotNil(t, delay)
	assert.Equal(t, 150*time.Second, *delay)
}

func TestParseRetryDelayBackoffDirective(t *testing.T) {
	body := []byte(`{"error": {"message": "backoff for 60s"}}`)
	delay, err := parseRetryDelay(body)
	assert.NoError(t, err)
	assert.NotNil(t, delay)
	assert.Equal(t, 60*time.Second, *delay)
}

func TestParseRetryDelayQuotaText(t *testing.T) {
	body := []byte(`{"error": {"message": "quota will reset in 30 seconds"}}`)
	delay, err := parseRetryDelay(body)
	assert.NoError(t, err)
	assert.NotNil(t, delay)
	assert.Equal(t, 30*time.Second, *delay)
}

func TestParseRetryDelayRetryAfterText(t *testing.T) {
	body := []byte(`{"error": {"message": "retry after 45 seconds"}}`)
	delay, err := parseRetryDelay(body)
	assert.NoError(t, err)
	assert.NotNil(t, delay)
	assert.Equal(t, 45*time.Second, *delay)
}

func TestParseDurationString(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
		wantErr  bool
	}{
		{"2h1m1s", 2*time.Hour + 1*time.Minute + 1*time.Second, false},
		{"500ms", 500 * time.Millisecond, false},
		{"42s", 42 * time.Second, false},
		{"2h", 2 * time.Hour, false},
		{"90m", 90 * time.Minute, false},
		{"invalid", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			d, err := parseDurationString(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, d)
			}
		})
	}
}

func TestParseRetryAfterHeader(t *testing.T) {
	tests := []struct {
		input    string
		expected *time.Duration
	}{
		{"", nil},
		{"invalid", nil},
		{"42", ptrDuration(42 * time.Second)},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			d := parseRetryAfterHeader(tt.input)
			if tt.expected == nil {
				assert.Nil(t, d)
			} else {
				assert.NotNil(t, d)
				assert.Equal(t, *tt.expected, *d)
			}
		})
	}
}

func ptrDuration(d time.Duration) *time.Duration {
	return &d
}

func TestParseRetryDelayUnknown(t *testing.T) {
	body := []byte(`{"error": {"message": "Unknown error"}}`)
	delay, err := parseRetryDelay(body)
	assert.Error(t, err)
	assert.Nil(t, delay)
}

func TestParseRetryDelaySafetyMinimum(t *testing.T) {
	body := []byte(`{"error": {"message": "after 1s."}}`)
	delay, err := parseRetryDelay(body)
	assert.NoError(t, err)
	assert.NotNil(t, delay)
	assert.Equal(t, 2*time.Second, *delay)
}

func TestParseRateLimitReasonJSON(t *testing.T) {
	tests := []struct {
		reason   string
		expected antigravity.RateLimitReason
	}{
		{"QUOTA_EXHAUSTED", antigravity.QuotaExhausted},
		{"RATE_LIMIT_EXCEEDED", antigravity.RateLimitExceeded},
		{"MODEL_CAPACITY_EXHAUSTED", antigravity.ModelCapacityExhausted},
	}

	for _, tt := range tests {
		t.Run(tt.reason, func(t *testing.T) {
			body := []byte(`{
				"error": {
					"details": [
						{
							"@type": "type.googleapis.com/google.rpc.ErrorInfo",
							"reason": "` + tt.reason + `"
						}
					]
				}
			}`)
			reason := parseRateLimitReason(body)
			assert.Equal(t, tt.expected, reason)
		})
	}
}

func TestParseRateLimitReasonTextFallback(t *testing.T) {
	tests := []struct {
		body     string
		expected antigravity.RateLimitReason
	}{
		{`{"error": {"message": "requests per minute exceeded"}}`, antigravity.RateLimitExceeded},
		{`{"error": {"message": "requests per second exceeded"}}`, antigravity.RateLimitExceeded},
		{`{"error": {"message": "model capacity exhausted"}}`, antigravity.ModelCapacityExhausted},
		{`{"error": {"message": "model overloaded"}}`, antigravity.ModelCapacityExhausted},
		{`{"error": {"message": "quota exceeded"}}`, antigravity.QuotaExhausted},
		{`{"error": {"message": "resource exhausted"}}`, antigravity.QuotaExhausted},
	}

	for _, tt := range tests {
		t.Run(tt.body, func(t *testing.T) {
			reason := parseRateLimitReason([]byte(tt.body))
			assert.Equal(t, tt.expected, reason)
		})
	}
}

func TestParseRateLimitReasonUnknown(t *testing.T) {
	body := []byte(`{"error": {"message": "Unknown error"}}`)
	reason := parseRateLimitReason(body)
	assert.Equal(t, antigravity.Unknown, reason)
}
