package management

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	claudeauth "github.com/router-for-me/CLIProxyAPI/v6/internal/auth/claude"
	coreauth "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/auth"
)

func TestGetClaudeQuotaNilHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/v0/management/claude-quota", nil)

	// nil handler — must not panic, returns 200 with empty array.
	var h *Handler
	h.GetClaudeQuota(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var result []interface{}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
	assert.Empty(t, result)
}

func TestGetClaudeQuotaNoAuthManager(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/v0/management/claude-quota", nil)

	// Handler with nil authManager → returns [] with 200.
	h := &Handler{}
	h.GetClaudeQuota(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var result []interface{}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
	assert.Empty(t, result)
}

func TestGetClaudeQuotaNoCLaudeAccounts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/v0/management/claude-quota", nil)

	store := &memoryAuthStore{}
	manager := coreauth.NewManager(store, nil, nil)

	// Register a non-claude auth — should be ignored.
	auth := &coreauth.Auth{
		ID:       "antigravity-account.json",
		FileName: "antigravity-account.json",
		Provider: "antigravity",
		Metadata: map[string]any{
			"access_token": "some-token",
		},
	}
	if _, err := manager.Register(context.Background(), auth); err != nil {
		t.Fatalf("register auth: %v", err)
	}

	h := &Handler{authManager: manager}
	h.GetClaudeQuota(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var result []interface{}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
	assert.Empty(t, result)
}

func TestGetClaudeQuotaWithValidAccount(t *testing.T) {
	gin.SetMode(gin.TestMode)

	resetsAt := time.Now().Add(2 * time.Hour).UTC().Truncate(time.Second)

	// Mock Anthropic usage API server.
	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Contains(t, r.Header.Get("Authorization"), "Bearer ")
		assert.Equal(t, "oauth-2025-04-20", r.Header.Get("anthropic-beta"))

		payload := map[string]any{
			"five_hour": map[string]any{
				"utilization": 30.0,
				"resets_at":   resetsAt.Format(time.RFC3339),
			},
			"seven_day": map[string]any{
				"utilization": 55.5,
				"resets_at":   resetsAt.Add(5 * 24 * time.Hour).Format(time.RFC3339),
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(payload)
	}))
	t.Cleanup(apiSrv.Close)

	// Override the Claude usage URL to redirect to the mock server.
	originalURL := claudeauth.ClaudeUsageURL
	claudeauth.ClaudeUsageURL = apiSrv.URL
	t.Cleanup(func() { claudeauth.ClaudeUsageURL = originalURL })

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/v0/management/claude-quota", nil)

	store := &memoryAuthStore{}
	manager := coreauth.NewManager(store, nil, nil)

	auth := &coreauth.Auth{
		ID:       "claude-test@example.com.json",
		FileName: "claude-test@example.com.json",
		Provider: "claude",
		Metadata: map[string]any{
			"access_token": "valid-access-token",
			"email":        "test@example.com",
			"type":         "claude",
		},
	}
	if _, err := manager.Register(context.Background(), auth); err != nil {
		t.Fatalf("register auth: %v", err)
	}

	h := &Handler{authManager: manager}
	h.GetClaudeQuota(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var results []claudeQuotaAccountEntry
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &results))
	assert.Len(t, results, 1)

	entry := results[0]
	assert.Equal(t, "claude-test@example.com.json", entry.AccountID)
	assert.Equal(t, "test@example.com", entry.Email)
	assert.InDelta(t, 30.0, entry.FiveHour.Utilization, 0.001)
	assert.InDelta(t, 70.0, entry.FiveHour.RemainingPercent, 0.001)
	assert.InDelta(t, 55.5, entry.SevenDay.Utilization, 0.001)
	assert.InDelta(t, 44.5, entry.SevenDay.RemainingPercent, 0.001)
}

func TestGetClaudeQuotaMissingEmail(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Mock API returning valid quota data.
	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		payload := map[string]any{
			"five_hour": map[string]any{
				"utilization": 10.0,
				"resets_at":   time.Now().Add(1 * time.Hour).Format(time.RFC3339),
			},
			"seven_day": map[string]any{
				"utilization": 20.0,
				"resets_at":   time.Now().Add(6 * 24 * time.Hour).Format(time.RFC3339),
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(payload)
	}))
	t.Cleanup(apiSrv.Close)

	originalURL := claudeauth.ClaudeUsageURL
	claudeauth.ClaudeUsageURL = apiSrv.URL
	t.Cleanup(func() { claudeauth.ClaudeUsageURL = originalURL })

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/v0/management/claude-quota", nil)

	store := &memoryAuthStore{}
	manager := coreauth.NewManager(store, nil, nil)

	// Auth without email — older auth file format.
	auth := &coreauth.Auth{
		ID:       "claude-no-email.json",
		FileName: "claude-no-email.json",
		Provider: "claude",
		Metadata: map[string]any{
			"access_token": "valid-token",
		},
	}
	if _, err := manager.Register(context.Background(), auth); err != nil {
		t.Fatalf("register auth: %v", err)
	}

	h := &Handler{authManager: manager}
	h.GetClaudeQuota(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var results []claudeQuotaAccountEntry
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &results))
	assert.Len(t, results, 1)
	assert.Equal(t, "", results[0].Email)
	assert.Equal(t, "claude-no-email.json", results[0].AccountID)
}
