// Package management provides the management API handlers and middleware
// for configuring the server and managing auth files.
package management

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	claudeauth "github.com/router-for-me/CLIProxyAPI/v6/internal/auth/claude"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/util"
)

// claudeUsageWindowEntry is the per-window quota data in the Claude API response.
type claudeUsageWindowEntry struct {
	Utilization      float64   `json:"utilization"`
	RemainingPercent float64   `json:"remaining_percent"`
	ResetsAt         time.Time `json:"resets_at"`
}

// claudeQuotaAccountEntry is the per-account Claude quota data in the API response.
type claudeQuotaAccountEntry struct {
	AccountID string                 `json:"account_id"`
	Email     string                 `json:"email"`
	FiveHour  claudeUsageWindowEntry `json:"five_hour"`
	SevenDay  claudeUsageWindowEntry `json:"seven_day"`
}

// GetClaudeQuota returns quota data for all Claude OAuth accounts.
//
// It iterates over all registered auth records, filters for the Claude provider,
// fetches live quota via the Anthropic usage API, and returns a JSON array of
// per-account quota entries. Falls back to cached data on fetch failure.
// Returns an empty array (not an error) if no Claude accounts are configured.
func (h *Handler) GetClaudeQuota(c *gin.Context) {
	results := make([]claudeQuotaAccountEntry, 0)

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

	checker := claudeauth.NewClaudeQuotaChecker(httpClient)
	ctx := c.Request.Context()

	// 3. Iterate over all auth records and collect Claude entries.
	auths := h.authManager.List()
	for _, auth := range auths {
		if auth.Provider != "claude" {
			continue
		}

		accessToken, _ := auth.Metadata["access_token"].(string)
		email, _ := auth.Metadata["email"].(string)
		accountID := auth.ID

		// 4. Fetch live quota — fall back to cached data on failure.
		quotaData, errFetch := checker.FetchQuota(ctx, accessToken, accountID)
		if errFetch != nil {
			quotaData = checker.GetCachedQuota(accountID)
		}
		if quotaData == nil {
			continue
		}

		// 5. Build the response entry from FiveHour and SevenDay windows.
		results = append(results, claudeQuotaAccountEntry{
			AccountID: accountID,
			Email:     email,
			FiveHour: claudeUsageWindowEntry{
				Utilization:      quotaData.FiveHour.Utilization,
				RemainingPercent: quotaData.FiveHour.RemainingPercent(),
				ResetsAt:         quotaData.FiveHour.ResetsAt,
			},
			SevenDay: claudeUsageWindowEntry{
				Utilization:      quotaData.SevenDay.Utilization,
				RemainingPercent: quotaData.SevenDay.RemainingPercent(),
				ResetsAt:         quotaData.SevenDay.ResetsAt,
			},
		})
	}

	// 6. Return JSON array — empty array when no Claude accounts found.
	c.JSON(http.StatusOK, results)
}
