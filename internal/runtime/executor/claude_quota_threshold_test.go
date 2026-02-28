package executor

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func TestQuotaThresholdError_StatusCode(t *testing.T) {
	err := &quotaThresholdError{
		model:       "claude-3-5-sonnet-20241022",
		provider:    "claude",
		utilization: 85.0,
		threshold:   80.0,
		resetsAt:    time.Now().Add(5 * time.Minute),
	}

	if got := err.StatusCode(); got != http.StatusTooManyRequests {
		t.Fatalf("StatusCode() = %d, want %d", got, http.StatusTooManyRequests)
	}
}

func TestQuotaThresholdError_RetryAfter(t *testing.T) {
	now := time.Now()
	resetsAt := now.Add(5 * time.Minute)
	err := &quotaThresholdError{
		model:       "claude-3-5-sonnet-20241022",
		provider:    "claude",
		utilization: 85.0,
		threshold:   80.0,
		resetsAt:    resetsAt,
	}

	retryAfter := err.RetryAfter()
	if retryAfter == nil {
		t.Fatal("RetryAfter() returned nil, want *time.Duration")
	}

	// Check that it's approximately 5 minutes (within 2 seconds tolerance)
	expectedDuration := time.Until(resetsAt)
	diff := *retryAfter - expectedDuration
	if diff < 0 {
		diff = -diff
	}
	tolerance := 2 * time.Second
	if diff > tolerance {
		t.Fatalf(
			"RetryAfter() = %v, want ~%v (tolerance: %v, diff: %v)",
			*retryAfter,
			expectedDuration,
			tolerance,
			diff,
		)
	}

	// Check that it's positive
	if *retryAfter <= 0 {
		t.Fatalf("RetryAfter() = %v, want > 0", *retryAfter)
	}
}

func TestQuotaThresholdError_RetryAfterClamped(t *testing.T) {
	// Past reset time should clamp to 0
	now := time.Now()
	resetsAt := now.Add(-1 * time.Minute) // 1 minute in the past
	err := &quotaThresholdError{
		model:       "claude-3-5-sonnet-20241022",
		provider:    "claude",
		utilization: 85.0,
		threshold:   80.0,
		resetsAt:    resetsAt,
	}

	retryAfter := err.RetryAfter()
	if retryAfter == nil {
		t.Fatal("RetryAfter() returned nil, want *time.Duration")
	}

	// Should be clamped to 0 (or very close to 0, not negative)
	if *retryAfter < 0 {
		t.Fatalf("RetryAfter() = %v, want >= 0 (clamped)", *retryAfter)
	}
}

func TestQuotaThresholdError_Error(t *testing.T) {
	now := time.Now()
	resetsAt := now.Add(2*time.Minute + 30*time.Second)
	err := &quotaThresholdError{
		model:       "claude-3-5-sonnet-20241022",
		provider:    "claude",
		utilization: 85.0,
		threshold:   80.0,
		resetsAt:    resetsAt,
	}

	errorStr := err.Error()

	// Parse the JSON response
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(errorStr), &payload); err != nil {
		t.Fatalf("Error() returned invalid JSON: %v", err)
	}

	// Check structure: {"error": {...}}
	errorBody, ok := payload["error"].(map[string]interface{})
	if !ok {
		t.Fatalf("Error() JSON missing 'error' field or it's not a map")
	}

	// Check required fields
	code, ok := errorBody["code"].(string)
	if !ok || code != "quota_threshold_exceeded" {
		t.Fatalf("error.code = %v, want 'quota_threshold_exceeded'", code)
	}

	message, ok := errorBody["message"].(string)
	if !ok || message == "" {
		t.Fatalf("error.message = %v, want non-empty string", message)
	}

	model, ok := errorBody["model"].(string)
	if !ok || model != "claude-3-5-sonnet-20241022" {
		t.Fatalf("error.model = %v, want 'claude-3-5-sonnet-20241022'", model)
	}

	utilization, ok := errorBody["utilization"].(float64)
	if !ok || utilization != 85.0 {
		t.Fatalf("error.utilization = %v, want 85.0", utilization)
	}

	threshold, ok := errorBody["threshold"].(float64)
	if !ok || threshold != 80.0 {
		t.Fatalf("error.threshold = %v, want 80.0", threshold)
	}

	resetTime, ok := errorBody["reset_time"].(string)
	if !ok || resetTime == "" {
		t.Fatalf("error.reset_time = %v, want non-empty string", resetTime)
	}

	resetSeconds, ok := errorBody["reset_seconds"].(float64)
	if !ok || resetSeconds < 0 {
		t.Fatalf("error.reset_seconds = %v, want >= 0", resetSeconds)
	}
}

func TestQuotaThresholdError_Headers(t *testing.T) {
	err := &quotaThresholdError{
		model:       "claude-3-5-sonnet-20241022",
		provider:    "claude",
		utilization: 85.0,
		threshold:   80.0,
		resetsAt:    time.Now().Add(3 * time.Minute),
	}

	headers := err.Headers()

	// Check Content-Type
	if contentType := headers.Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("Content-Type = %s, want 'application/json'", contentType)
	}

	// Check Retry-After is present and numeric
	retryAfter := headers.Get("Retry-After")
	if retryAfter == "" {
		t.Fatal("Retry-After header not set")
	}

	// Should be numeric seconds (just verify non-empty)
	if retryAfter == "" {
		t.Fatal("Retry-After header is empty")
	}
}
