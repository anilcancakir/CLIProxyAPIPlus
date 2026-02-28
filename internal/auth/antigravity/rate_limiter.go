package antigravity

import (
"regexp"
	"strconv"
	"strings"
"sync"
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

type rateLimitEntry struct {
	info      *RateLimitInfo
	expiresAt time.Time
}

type failureEntry struct {
	count       int
	lastFailure time.Time
}

type AntigravityRateLimiter struct {
	mu           sync.RWMutex
	limits       map[string]*rateLimitEntry
	failures     map[string]*failureEntry
	backoffSteps []time.Duration
}

var defaultBackoffSteps = []time.Duration{
	60 * time.Second,
	300 * time.Second,
	1800 * time.Second,
	7200 * time.Second,
}

func NewAntigravityRateLimiter() *AntigravityRateLimiter {
	return NewAntigravityRateLimiterWithSteps(defaultBackoffSteps)
}

func NewAntigravityRateLimiterWithSteps(steps []time.Duration) *AntigravityRateLimiter {
	return &AntigravityRateLimiter{
		limits:       make(map[string]*rateLimitEntry),
		failures:     make(map[string]*failureEntry),
		backoffSteps: steps,
	}
}

func getLimitKey(accountID string, reason RateLimitReason, model *string) string {
	if reason == QuotaExhausted && model != nil {
		return accountID + ":" + *model
	}
	return accountID
}

func (rl *AntigravityRateLimiter) ParseFromError(
accountID string,
status int,
retryAfterHeader *string,
body []byte,
model *string,
) *RateLimitInfo {
	var reason RateLimitReason = Unknown
	lowerBody := strings.ToLower(string(body))
	if strings.Contains(lowerBody, "quota exhausted") {
		reason = QuotaExhausted
	} else if strings.Contains(lowerBody, "rate limit exceeded") {
		reason = RateLimitExceeded
	} else if strings.Contains(lowerBody, "model capacity") {
		reason = ModelCapacityExhausted
	}

	isServerError := status >= 500 || status == 404
	if isServerError {
		reason = ServerError
	}


	var duration time.Duration

	if retryAfterHeader != nil && *retryAfterHeader != "" {
		if secs, err := strconv.ParseUint(*retryAfterHeader, 10, 64); err == nil {
			duration = time.Duration(secs) * time.Second
		}
	}

	if duration == 0 {
		duration = extractDurationFromBody(body)
	}

	if duration == 0 {
		rl.mu.RLock()
		duration = rl.defaultDurationForReason(reason, accountID)
		rl.mu.RUnlock()
	}

	if duration < 2*time.Second {
		duration = 2 * time.Second
	}

	info := &RateLimitInfo{
		Reason:        reason,
		RetryAfterSec: uint64(duration.Seconds()),
		DetectedAt:    time.Now(),
		Model:         model,
		ResetTime:     time.Now().Add(duration),
	}

	key := getLimitKey(accountID, reason, model)
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.limits[key] = &rateLimitEntry{
		info:      info,
		expiresAt: time.Now().Add(duration),
	}

	if !isServerError {
		rl.incrementFailureCount(accountID)
	}

	return info
}

func (rl *AntigravityRateLimiter) defaultDurationForReason(reason RateLimitReason, accountID string) time.Duration {
	switch reason {
	case QuotaExhausted:
		failCount := rl.getFailureCount(accountID)
		idx := failCount
		if idx >= len(rl.backoffSteps) {
			idx = len(rl.backoffSteps) - 1
		}
		return rl.backoffSteps[idx]
	case RateLimitExceeded:
		return 5 * time.Second
	case ModelCapacityExhausted:
		failCount := rl.getFailureCount(accountID)
		steps := []time.Duration{5 * time.Second, 10 * time.Second, 15 * time.Second}
		idx := failCount
		if idx >= len(steps) {
			idx = len(steps) - 1
		}
		return steps[idx]
	case ServerError:
		return 8 * time.Second
	default:
		return 60 * time.Second
	}
}

func extractDurationFromBody(body []byte) time.Duration {
	s := string(body)
	if strings.Contains(strings.ToLower(s), "quota exhausted") {
		return 0
	}
re := regexp.MustCompile(`(\d+)m\s*(\d+)s`)
	if m := re.FindStringSubmatch(s); len(m) == 3 {
		mins, _ := strconv.Atoi(m[1])
		secs, _ := strconv.Atoi(m[2])
		return time.Duration(mins*60+secs) * time.Second
	}
	re2 := regexp.MustCompile(`after\s+(\d+)s`)
	if m := re2.FindStringSubmatch(s); len(m) == 2 {
		secs, _ := strconv.Atoi(m[1])
		return time.Duration(secs) * time.Second
	}
	re3 := regexp.MustCompile(`(\d+)\s*second`)
	if m := re3.FindStringSubmatch(s); len(m) == 2 {
		secs, _ := strconv.Atoi(m[1])
		return time.Duration(secs) * time.Second
	}
	return 0
}

func (rl *AntigravityRateLimiter) incrementFailureCount(accountID string) {
	entry := rl.failures[accountID]
	if entry == nil {
		entry = &failureEntry{}
		rl.failures[accountID] = entry
	}
	if !entry.lastFailure.IsZero() && time.Since(entry.lastFailure) > time.Hour {
		entry.count = 0
	}
	entry.count++
	entry.lastFailure = time.Now()
}

func (rl *AntigravityRateLimiter) getFailureCount(accountID string) int {
	entry := rl.failures[accountID]
	if entry == nil {
		return 0
	}
	if !entry.lastFailure.IsZero() && time.Since(entry.lastFailure) > time.Hour {
		return 0
	}
	return entry.count
}

func (rl *AntigravityRateLimiter) IsRateLimited(accountID string, model *string) bool {
	rl.mu.RLock()
	defer rl.mu.RUnlock()
	now := time.Now()

	if entry, ok := rl.limits[accountID]; ok && entry.expiresAt.After(now) {
		return true
	}

	if model != nil {
		modelKey := accountID + ":" + *model
		if entry, ok := rl.limits[modelKey]; ok && entry.expiresAt.After(now) {
			return true
		}
	}
	return false
}

func (rl *AntigravityRateLimiter) GetRemainingWait(accountID string, model *string) time.Duration {
	rl.mu.RLock()
	defer rl.mu.RUnlock()
	now := time.Now()
	var longest time.Duration

	if entry, ok := rl.limits[accountID]; ok && entry.expiresAt.After(now) {
		if d := entry.expiresAt.Sub(now); d > longest {
			longest = d
		}
	}
	if model != nil {
		modelKey := accountID + ":" + *model
		if entry, ok := rl.limits[modelKey]; ok && entry.expiresAt.After(now) {
			if d := entry.expiresAt.Sub(now); d > longest {
				longest = d
			}
		}
	}
	return longest
}

func (rl *AntigravityRateLimiter) MarkSuccess(accountID string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.failures, accountID)
	delete(rl.limits, accountID)
}

func (rl *AntigravityRateLimiter) SetLockoutUntil(accountID string, resetTime time.Time, reason RateLimitReason, model *string) {
	key := getLimitKey(accountID, reason, model)
	info := &RateLimitInfo{
		Reason:        reason,
		ResetTime:     resetTime,
		RetryAfterSec: uint64(time.Until(resetTime).Seconds()),
		DetectedAt:    time.Now(),
		Model:         model,
	}
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.limits[key] = &rateLimitEntry{
		info:      info,
		expiresAt: resetTime,
	}
}

func (rl *AntigravityRateLimiter) CleanupExpired() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()
	for k, v := range rl.limits {
		if v.expiresAt.Before(now) {
			delete(rl.limits, k)
		}
	}
}
