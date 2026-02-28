package antigravity

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestModelQuota(t *testing.T) {
	t.Run("TestModelQuotaSerialization", func(t *testing.T) {
		now := time.Now().Round(time.Second)
		quota := ModelQuota{
			Name:              "gemini-1.5-pro",
			RemainingFraction: 0.75,
			ResetTime:         now,
			DisplayName:       "Gemini 1.5 Pro",
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
	})

	t.Run("TestRemainingPercent", func(t *testing.T) {
		quota := ModelQuota{
			RemainingFraction: 0.75,
		}
		assert.Equal(t, 75.0, quota.RemainingPercent())

		quotaZero := ModelQuota{
			RemainingFraction: 0.0,
		}
		assert.Equal(t, 0.0, quotaZero.RemainingPercent())
	})
}

func TestQuotaData(t *testing.T) {
	t.Run("TestQuotaDataEmpty", func(t *testing.T) {
		qd := QuotaData{
			LastUpdated: time.Now(),
		}

		assert.Empty(t, qd.Models)

		quota := ModelQuota{
			Name: "gemini-1.5-flash",
		}

		qd.AddModel(quota)
		assert.Len(t, qd.Models, 1)
		assert.Equal(t, "gemini-1.5-flash", qd.Models[0].Name)
	})

	t.Run("TestQuotaDataSerialization", func(t *testing.T) {
		now := time.Now().Round(time.Second)
		qd := QuotaData{
			Models: []ModelQuota{
				{
					Name:              "gemini-1.5-pro",
					RemainingFraction: 0.5,
					ResetTime:         now,
					DisplayName:       "Gemini 1.5 Pro",
				},
			},
			LastUpdated: now,
			IsForbidden: true,
		}

		data, err := json.Marshal(qd)
		assert.NoError(t, err)

		var unmarshaled QuotaData
		err = json.Unmarshal(data, &unmarshaled)
		assert.NoError(t, err)

		assert.Len(t, unmarshaled.Models, 1)
		assert.Equal(t, qd.Models[0].Name, unmarshaled.Models[0].Name)
		assert.True(t, qd.LastUpdated.Equal(unmarshaled.LastUpdated))
		assert.True(t, qd.IsForbidden == unmarshaled.IsForbidden)
	})
}
