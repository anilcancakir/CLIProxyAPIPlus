package antigravity

import (
	"context"
	"encoding/json"
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
