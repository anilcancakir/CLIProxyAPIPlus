package codex

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// TestRefreshTokensWithRetry_ReturnsErrorAfterMaxRetries verifies that RefreshTokensWithRetry
// returns an error and exhausts all retry attempts before giving up.
// NOTE: Origin's implementation retries all maxRetries times regardless of error type.
// The fork added early-exit for specific error codes; that behaviour is not in origin.
func TestRefreshTokensWithRetry_ReturnsErrorAfterMaxRetries(t *testing.T) {
	const maxRetries = 2
	var calls int32
	auth := &CodexAuth{
		httpClient: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				atomic.AddInt32(&calls, 1)
				return &http.Response{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader(`{"error":"internal_server_error"}`)),

					Header:     make(http.Header),
					Request:    req,
				}, nil
			}),
		},
	}

	_, err := auth.RefreshTokensWithRetry(context.Background(), "dummy_refresh_token", maxRetries)
	if err == nil {
		t.Fatalf("expected error for failed refresh, got nil")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "400") {
		t.Fatalf("expected HTTP 400 status in error, got: %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != maxRetries {
		t.Fatalf("expected %d refresh attempts (maxRetries), got %d", maxRetries, got)
	}
}
