// Package management provides the management API handlers and middleware
// for configuring the server and managing auth files.
package management

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/auth/antigravity"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/util"
)

// quotaModelEntry is the per-model quota data in the API response.
type quotaModelEntry struct {
	Name              string    `json:"name"`
	DisplayName       string    `json:"display_name"`
	RemainingFraction float64   `json:"remaining_fraction"`
	RemainingPercent  float64   `json:"remaining_percent"`
	ResetTime         time.Time `json:"reset_time"`
}

// quotaAccountEntry is the per-account quota data in the API response.
type quotaAccountEntry struct {
	AccountID string            `json:"account_id"`
	Email     string            `json:"email"`
	Models    []quotaModelEntry `json:"models"`
}

// GetAntigravityQuota returns quota data for all Antigravity accounts.
//
// It iterates over all registered auth records, filters for Antigravity provider,
// fetches live quota via the fetchAvailableModels API, and returns a JSON array
// of per-account quota entries. Falls back to cached data on fetch failure.
// Returns an empty array (not an error) if no Antigravity accounts are configured.
func (h *Handler) GetAntigravityQuota(c *gin.Context) {
	results := make([]quotaAccountEntry, 0)

	// 1. Guard: handler or authManager unavailable — return empty array.
	if h == nil || h.authManager == nil {
		c.JSON(http.StatusOK, results)
		return
	}

	// 2. Build a proxy-aware HTTP client scoped to the configured proxy settings.
	var httpClient *http.Client
	if h.cfg != nil {
		httpClient = util.SetProxy(&h.cfg.SDKConfig, &http.Client{Timeout: 15 * time.Second})
	} else {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}

	checker := antigravity.NewAntigravityQuotaChecker(httpClient)
	ctx := c.Request.Context()

	// 3. Iterate over all auth records and collect Antigravity entries.
	auths := h.authManager.List()
	for _, auth := range auths {
		if auth.Provider != "antigravity" {
			continue
		}

		accessToken, _ := auth.Metadata["access_token"].(string)
		email, _ := auth.Metadata["email"].(string)
		accountID := auth.ID

		// 4. Fetch live quota — fall back to cached data on failure.
		quotaData, errFetch := checker.FetchQuota(ctx, accessToken, email, accountID)
		if errFetch != nil {
			quotaData = checker.GetCachedQuota(accountID)
		}
		if quotaData == nil {
			continue
		}

		// 5. Convert QuotaData.Models to quotaModelEntry slice.
		models := make([]quotaModelEntry, 0, len(quotaData.Models))
		for _, m := range quotaData.Models {
			models = append(models, quotaModelEntry{
				Name:              m.Name,
				DisplayName:       m.DisplayName,
				RemainingFraction: m.RemainingFraction,
				RemainingPercent:  m.RemainingPercent,
				ResetTime:         m.ResetTime,
			})
		}

		results = append(results, quotaAccountEntry{
			AccountID: accountID,
			Email:     email,
			Models:    models,
		})
	}

	// 6. Return JSON array — empty array when no Antigravity accounts found.
	c.JSON(http.StatusOK, results)
}
