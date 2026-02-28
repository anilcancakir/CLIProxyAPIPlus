package antigravity

import (
	"time"
)

// RateLimitReason represents the reason for a rate limit event.
type RateLimitReason int

const (
	// Unknown reason or not yet determined.
	Unknown RateLimitReason = iota
	// QuotaExhausted means the user's quota has been completely used up.
	QuotaExhausted
	// RateLimitExceeded means the requests per minute/second limit has been hit.
	RateLimitExceeded
	// ModelCapacityExhausted means the model is temporarily overloaded.
	ModelCapacityExhausted
	// ServerError means a generic server-side error occurred.
	ServerError
)

// String returns the string representation of the RateLimitReason.
func (r RateLimitReason) String() string {
	switch r {
	case Unknown:
		return "UNKNOWN"
	case QuotaExhausted:
		return "QUOTA_EXHAUSTED"
	case RateLimitExceeded:
		return "RATE_LIMIT_EXCEEDED"
	case ModelCapacityExhausted:
		return "MODEL_CAPACITY_EXHAUSTED"
	case ServerError:
		return "SERVER_ERROR"
	default:
		return "UNKNOWN"
	}
}

// RateLimitInfo contains detailed information about a rate limit event.
type RateLimitInfo struct {
	// Reason why the rate limit was triggered.
	Reason RateLimitReason `json:"reason"`
	// ResetTime is when the rate limit is expected to expire.
	ResetTime time.Time `json:"reset_time"`
	// RetryAfterSec is the number of seconds to wait before retrying.
	RetryAfterSec uint64 `json:"retry_after_sec"`
	// DetectedAt is when the rate limit was first detected.
	DetectedAt time.Time `json:"detected_at"`
	// Model is the specific model that was rate limited. Nil means account-level.
	Model *string `json:"model,omitempty"`
}

// IsModelLevel returns true if the rate limit is specific to a model.
func (i RateLimitInfo) IsModelLevel() bool {
	return i.Model != nil
}

// RateLimiter defines the interface for managing rate limits for the Antigravity provider.
type RateLimiter interface {
	// IsRateLimited checks if the given account and model are currently rate limited.
	IsRateLimited(accountID string, model *string) bool

	// GetRemainingWait returns the duration to wait before the next request is allowed.
	GetRemainingWait(accountID string, model *string) time.Duration

	// ParseFromError parses rate limit information from an API error response.
	ParseFromError(
		accountID string,
		status int,
		retryAfterHeader *string,
		body []byte,
		model *string,
	) *RateLimitInfo

	// MarkSuccess clears any transient rate limit state for the account.
	MarkSuccess(accountID string)

	// SetLockoutUntil manually sets a lockout period for an account or model.
	SetLockoutUntil(
		accountID string,
		resetTime time.Time,
		reason RateLimitReason,
		model *string,
	)
}
