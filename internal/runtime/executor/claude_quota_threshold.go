package executor

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"
)

// quotaThresholdError represents a quota utilization threshold exceeded error.
// It implements the retryAfterProvider interface for conductor auto-recovery.
type quotaThresholdError struct {
	model       string
	provider    string
	utilization float64
	threshold   float64
	resetsAt    time.Time
}

// Error returns a JSON-formatted error message with quota details.
// The error code is "quota_threshold_exceeded" to distinguish from model cooldown.
func (e *quotaThresholdError) Error() string {
	message := fmt.Sprintf(
		"Quota threshold exceeded for model %s (%.1f%% >= %.1f%% threshold)",
		e.model,
		e.utilization,
		e.threshold,
	)
	if e.provider != "" {
		message = fmt.Sprintf("%s via provider %s", message, e.provider)
	}

	// Calculate reset duration and display format
	resetDuration := time.Until(e.resetsAt)
	if resetDuration < 0 {
		resetDuration = 0
	}

	resetSeconds := int(math.Ceil(resetDuration.Seconds()))
	if resetSeconds < 0 {
		resetSeconds = 0
	}

	displayDuration := resetDuration
	if displayDuration > 0 && displayDuration < time.Second {
		displayDuration = time.Second
	} else {
		displayDuration = displayDuration.Round(time.Second)
	}

	errorBody := map[string]interface{}{
		"code":          "quota_threshold_exceeded",
		"message":       message,
		"model":         e.model,
		"utilization":   e.utilization,
		"threshold":     e.threshold,
		"reset_time":    displayDuration.String(),
		"reset_seconds": resetSeconds,
	}
	if e.provider != "" {
		errorBody["provider"] = e.provider
	}

	payload := map[string]interface{}{"error": errorBody}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Sprintf(
			`{"error":{"code":"quota_threshold_exceeded","message":"%s"}}`,
			message,
		)
	}
	return string(data)
}

// StatusCode returns HTTP 429 (Too Many Requests).
func (e *quotaThresholdError) StatusCode() int {
	return http.StatusTooManyRequests
}

// RetryAfter returns a pointer to the duration until quota resets.
// It computes time.Until(resetsAt) at call time, clamped to >= 0 for past times.
func (e *quotaThresholdError) RetryAfter() *time.Duration {
	duration := time.Until(e.resetsAt)
	if duration < 0 {
		duration = 0
	}
	return &duration
}

// Headers returns HTTP headers for the error response.
// Sets Content-Type to application/json and Retry-After to the reset duration in seconds.
func (e *quotaThresholdError) Headers() http.Header {
	headers := make(http.Header)
	headers.Set("Content-Type", "application/json")

	resetDuration := time.Until(e.resetsAt)
	if resetDuration < 0 {
		resetDuration = 0
	}

	resetSeconds := int(math.Ceil(resetDuration.Seconds()))
	if resetSeconds < 0 {
		resetSeconds = 0
	}

	headers.Set("Retry-After", strconv.Itoa(resetSeconds))
	return headers
}
