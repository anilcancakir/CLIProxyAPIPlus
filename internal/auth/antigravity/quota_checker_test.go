package antigravity

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestModelQuotaSerialization(t *testing.T) {
	now := time.Now().Round(time.Second)
	quota := ModelQuota{
		Name:              "gemini-1.5-pro",
		RemainingFraction: 0.75,
		ResetTime:         now,
		DisplayName:       "Gemini 1.5 Pro",
		RemainingPercent:  75.0,
	}

	// 1. Test JSON Marshalling
	data, err := json.Marshal(quota)
	assert.NoError(t, err)

	var unmarshaled ModelQuota
	err = json.Unmarshal(data, &unmarshaled)
	assert.NoError(t, err)

	// 2. Verify fields match
	assert.Equal(t, quota.Name, unmarshaled.Name)
	assert.Equal(t, quota.RemainingFraction, unmarshaled.RemainingFraction)
	assert.True(t, quota.ResetTime.Equal(unmarshaled.ResetTime))
	assert.Equal(t, quota.DisplayName, unmarshaled.DisplayName)
	assert.Equal(t, quota.RemainingPercent, unmarshaled.RemainingPercent)
}

func TestQuotaDataEmpty(t *testing.T) {
	qd := QuotaData{
		LastUpdated: time.Now(),
	}

	// 1. Assert Models slice is empty but not nil (after AddModel)
	assert.Empty(t, qd.Models)

	quota := ModelQuota{
		Name: "gemini-1.5-flash",
	}

	// 2. Test AddModel
	qd.AddModel(quota)
	assert.Len(t, qd.Models, 1)
	assert.Equal(t, "gemini-1.5-flash", qd.Models[0].Name)
}

func TestQuotaCheckerInterface(t *testing.T) {
	// compile-time check for interface compliance
	var _ QuotaChecker = (*mockQuotaChecker)(nil)
}

type mockQuotaChecker struct{}

func (m *mockQuotaChecker) FetchQuota(ctx context.Context, accessToken, email, accountID string) (*QuotaData, error) {
	return nil, nil
}

func TestFetchQuotaSuccess(t *testing.T) {
	// Setup mock server
	mockResponse := `{
		"models": {
			"gemini-2.5-pro": {
				"displayName": "Gemini 2.5 Pro",
				"quotaInfo": {
					"remainingFraction": 0.75,
					"resetTime": "2026-03-01T00:00:00Z"
				}
			},
			"gemini-2.5-flash": {
				"displayName": "Gemini 2.5 Flash",
				"quotaInfo": {
					"remainingFraction": 0.0,
					"resetTime": "2026-03-01T00:00:00Z"
				}
			}
		}
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1internal:fetchAvailableModels", r.URL.Path)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	checker := NewAntigravityQuotaChecker(server.Client())
	checker.baseURL = server.URL

	ctx := context.Background()
	quotaData, err := checker.FetchQuota(ctx, "test-token", "test@example.com", "acc-123")

	assert.NoError(t, err)
	assert.NotNil(t, quotaData)
	assert.False(t, quotaData.IsForbidden)
	assert.Len(t, quotaData.Models, 2)

	// Validate parsed models
	var proModel, flashModel ModelQuota
	for _, m := range quotaData.Models {
		if m.Name == "gemini-2.5-pro" {
			proModel = m
		} else if m.Name == "gemini-2.5-flash" {
			flashModel = m
		}
	}

	assert.Equal(t, "Gemini 2.5 Pro", proModel.DisplayName)
	assert.Equal(t, 0.75, proModel.RemainingFraction)
	assert.Equal(t, 75.0, proModel.RemainingPercent)
	expectedTime, _ := time.Parse(time.RFC3339, "2026-03-01T00:00:00Z")
	assert.True(t, expectedTime.Equal(proModel.ResetTime))

	assert.Equal(t, "Gemini 2.5 Flash", flashModel.DisplayName)
	assert.Equal(t, 0.0, flashModel.RemainingFraction)
	assert.Equal(t, 0.0, flashModel.RemainingPercent)
}

func TestFetchQuota403(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	checker := NewAntigravityQuotaChecker(server.Client())
	checker.baseURL = server.URL

	ctx := context.Background()
	quotaData, err := checker.FetchQuota(ctx, "test-token", "test@example.com", "acc-403")

	assert.NoError(t, err) // 403 should not return an error
	assert.NotNil(t, quotaData)
	assert.True(t, quotaData.IsForbidden)
	assert.Empty(t, quotaData.Models)
}

func TestGetCachedQuota(t *testing.T) {
	mockResponse := `{
		"models": {
			"gemini-2.5-pro": {
				"quotaInfo": {
					"remainingFraction": 1.0
				}
			}
		}
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	checker := NewAntigravityQuotaChecker(server.Client())
	checker.baseURL = server.URL

	ctx := context.Background()
	accountID := "acc-cache-test"

	// 1. Assert cache is initially empty
	assert.Nil(t, checker.GetCachedQuota(accountID))

	// 2. Fetch data
	fetchedQuota, err := checker.FetchQuota(ctx, "test-token", "test@example.com", accountID)
	assert.NoError(t, err)
	assert.NotNil(t, fetchedQuota)

	// 3. Assert cached data matches fetched data
	cachedQuota := checker.GetCachedQuota(accountID)
	assert.NotNil(t, cachedQuota)
	assert.Equal(t, fetchedQuota, cachedQuota)
	assert.Len(t, cachedQuota.Models, 1)
	assert.Equal(t, "gemini-2.5-pro", cachedQuota.Models[0].Name)
}

func TestQuotaCacheRace(t *testing.T) {
	mockResponse := `{
		"models": {
			"gemini-2.5-pro": {"quotaInfo": {"remainingFraction": 1.0}}
		}
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	checker := NewAntigravityQuotaChecker(server.Client())
	checker.baseURL = server.URL

	ctx := context.Background()
	accountID := "acc-race-test"

	// Launch multiple goroutines to test concurrent reads and writes
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(i int) {
			if i%2 == 0 {
				_, _ = checker.FetchQuota(ctx, "token", "email", accountID)
			} else {
				_ = checker.GetCachedQuota(accountID)
			}
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestFetchQuotaFiltersModels(t *testing.T) {
	mockResponse := `{
		"models": {
			"gemini-2.5-pro": {
				"quotaInfo": {"remainingFraction": 1.0}
			},
			"claude-3.5-sonnet": {
				"quotaInfo": {"remainingFraction": 1.0}
			},
			"internal-model-x": {
				"quotaInfo": {"remainingFraction": 1.0}
			},
			"gpt-4o": {
				"quotaInfo": {"remainingFraction": 1.0}
			}
		}
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	checker := NewAntigravityQuotaChecker(server.Client())
	checker.baseURL = server.URL

	ctx := context.Background()
	quotaData, err := checker.FetchQuota(ctx, "test-token", "test@example.com", "acc-filter")

	assert.NoError(t, err)
	assert.NotNil(t, quotaData)

	// Should only contain gemini, claude, gpt
	assert.Len(t, quotaData.Models, 3)

	names := make(map[string]bool)
	for _, m := range quotaData.Models {
		names[m.Name] = true
	}

	assert.True(t, names["gemini-2.5-pro"])
	assert.True(t, names["claude-3.5-sonnet"])
	assert.True(t, names["gpt-4o"])
	assert.False(t, names["internal-model-x"])
}
