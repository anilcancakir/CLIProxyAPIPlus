package claude

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newQuotaTestChecker(t *testing.T, server *httptest.Server) *ClaudeQuotaChecker {
	t.Helper()

	targetURL := server.URL
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		rewritten, err := http.NewRequestWithContext(
			req.Context(),
			req.Method,
			targetURL,
			req.Body,
		)
		if err != nil {
			return nil, err
		}

		rewritten.Header = req.Header.Clone()

		return http.DefaultTransport.RoundTrip(rewritten)
	})

	return NewClaudeQuotaChecker(&http.Client{
		Transport: transport,
	})
}

func TestClaudeQuotaCheckerFetchQuota(t *testing.T) {
	t.Run("Fresh fetch", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "Bearer test-access-token", r.Header.Get("Authorization"))
			assert.Equal(t, claudeOAuthBeta, r.Header.Get("anthropic-beta"))

			response := map[string]any{
				"five_hour": map[string]any{
					"utilization": 42.5,
					"resets_at":   "2026-01-01T12:00:00Z",
				},
				"seven_day": map[string]any{
					"utilization": 10.0,
					"resets_at":   "2026-01-07T00:00:00Z",
				},
			}

			require.NoError(t, json.NewEncoder(w).Encode(response))
		}))
		defer server.Close()

		checker := newQuotaTestChecker(t, server)

		data, err := checker.FetchQuota(
			context.Background(),
			"test-access-token",
			"account-1",
		)
		require.NoError(t, err)
		require.NotNil(t, data)

		assert.Equal(t, 42.5, data.FiveHour.Utilization)
		assert.Equal(t, 57.5, data.FiveHour.RemainingPercent())
		assert.Equal(t, 10.0, data.SevenDay.Utilization)
	})

	t.Run("Cache hit", func(t *testing.T) {
		var requestCount int32

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&requestCount, 1)

			response := map[string]any{
				"five_hour": map[string]any{
					"utilization": 21.0,
					"resets_at":   "2026-01-01T12:00:00Z",
				},
				"seven_day": map[string]any{
					"utilization": 8.0,
					"resets_at":   "2026-01-07T00:00:00Z",
				},
			}

			require.NoError(t, json.NewEncoder(w).Encode(response))
		}))
		defer server.Close()

		checker := newQuotaTestChecker(t, server)

		first, err := checker.FetchQuota(
			context.Background(),
			"token",
			"cache-account",
		)
		require.NoError(t, err)
		require.NotNil(t, first)

		second, err := checker.FetchQuota(
			context.Background(),
			"token",
			"cache-account",
		)
		require.NoError(t, err)
		require.NotNil(t, second)

		assert.Equal(t, int32(1), atomic.LoadInt32(&requestCount))
	})

	t.Run("HTTP 429", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTooManyRequests)
		}))
		defer server.Close()

		checker := newQuotaTestChecker(t, server)

		data, err := checker.FetchQuota(
			context.Background(),
			"token",
			"limited-account",
		)
		require.Error(t, err)
		assert.Nil(t, data)
		assert.Contains(t, err.Error(), "rate limited")
	})

	t.Run("HTTP 500", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, writeErr := w.Write([]byte("internal error"))
			require.NoError(t, writeErr)
		}))
		defer server.Close()

		checker := newQuotaTestChecker(t, server)

		data, err := checker.FetchQuota(
			context.Background(),
			"token",
			"error-account",
		)
		require.Error(t, err)
		assert.Nil(t, data)
	})

	t.Run("Empty response fields", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, writeErr := w.Write([]byte("{}"))
			require.NoError(t, writeErr)
		}))
		defer server.Close()

		checker := newQuotaTestChecker(t, server)

		data, err := checker.FetchQuota(
			context.Background(),
			"token",
			"empty-account",
		)
		require.NoError(t, err)
		require.NotNil(t, data)

		assert.Equal(t, 0.0, data.FiveHour.Utilization)
		assert.Equal(t, 0.0, data.SevenDay.Utilization)
		assert.True(t, data.FiveHour.ResetsAt.IsZero())
		assert.True(t, data.SevenDay.ResetsAt.IsZero())
	})
}

func TestClaudeQuotaCheckerGetCachedQuota(t *testing.T) {
	t.Run("GetCachedQuota nil", func(t *testing.T) {
		checker := NewClaudeQuotaChecker(&http.Client{
			Timeout: 5 * time.Second,
		})

		cached := checker.GetCachedQuota("unknown")
		assert.Nil(t, cached)
	})
}
