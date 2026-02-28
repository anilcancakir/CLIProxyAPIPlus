package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

var (
	// ClaudeUsageURL is the Anthropic OAuth usage endpoint. Overridable in tests.
	ClaudeUsageURL  = "https://api.anthropic.com/api/oauth/usage"
	claudeOAuthBeta = "oauth-2025-04-20"
)

const claudeQuotaTTL = 5 * time.Minute

type ClaudeUsageWindow struct {
	Utilization float64
	ResetsAt    time.Time
}

func (w ClaudeUsageWindow) RemainingPercent() float64 {
	return 100 - w.Utilization
}

func (w ClaudeUsageWindow) RemainingFraction() float64 {
	return (100 - w.Utilization) / 100
}

type ClaudeQuotaData struct {
	FiveHour    ClaudeUsageWindow
	SevenDay    ClaudeUsageWindow
	LastUpdated time.Time
}

type cachedClaudeQuota struct {
	data      *ClaudeQuotaData
	fetchedAt time.Time
}

type ClaudeQuotaChecker struct {
	httpClient *http.Client
	mu         sync.RWMutex
	cache      map[string]*cachedClaudeQuota
}

func NewClaudeQuotaChecker(httpClient *http.Client) *ClaudeQuotaChecker {
	return &ClaudeQuotaChecker{
		httpClient: httpClient,
		cache:      make(map[string]*cachedClaudeQuota),
	}
}

func (q *ClaudeQuotaChecker) FetchQuota(
	ctx context.Context,
	accessToken string,
	accountID string,
) (*ClaudeQuotaData, error) {
	q.mu.RLock()
	cached := q.cache[accountID]
	q.mu.RUnlock()

	if cached != nil && time.Since(cached.fetchedAt) < claudeQuotaTTL {
		return cached.data, nil
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		ClaudeUsageURL,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create claude quota request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("anthropic-beta", claudeOAuthBeta)

	resp, err := q.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute claude quota request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("claude quota rate limited for account %s", accountID)
	}

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, fmt.Errorf(
				"claude quota request failed for account %s with status %d",
				accountID,
				resp.StatusCode,
			)
		}

		return nil, fmt.Errorf(
			"claude quota request failed for account %s with status %d: %s",
			accountID,
			resp.StatusCode,
			strings.TrimSpace(string(body)),
		)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read claude quota response body: %w", err)
	}

	type usageWindowPayload struct {
		Utilization *float64 `json:"utilization"`
		ResetsAt    string   `json:"resets_at"`
	}

	type quotaPayload struct {
		FiveHour usageWindowPayload `json:"five_hour"`
		SevenDay usageWindowPayload `json:"seven_day"`
	}

	payload := quotaPayload{}
	if len(body) > 0 {
		if err := json.Unmarshal(body, &payload); err != nil {
			return nil, fmt.Errorf("failed to parse claude quota response: %w", err)
		}
	}

	data := &ClaudeQuotaData{
		FiveHour: ClaudeUsageWindow{
			Utilization: valueOrZero(payload.FiveHour.Utilization),
			ResetsAt:    parseRFC3339OrZero(payload.FiveHour.ResetsAt),
		},
		SevenDay: ClaudeUsageWindow{
			Utilization: valueOrZero(payload.SevenDay.Utilization),
			ResetsAt:    parseRFC3339OrZero(payload.SevenDay.ResetsAt),
		},
		LastUpdated: time.Now(),
	}

	q.mu.Lock()
	q.cache[accountID] = &cachedClaudeQuota{
		data:      data,
		fetchedAt: time.Now(),
	}
	q.mu.Unlock()

	return data, nil
}

func (q *ClaudeQuotaChecker) GetCachedQuota(accountID string) *ClaudeQuotaData {
	q.mu.RLock()
	defer q.mu.RUnlock()

	cached := q.cache[accountID]
	if cached == nil {
		return nil
	}

	return cached.data
}

func valueOrZero(value *float64) float64 {
	if value == nil {
		return 0
	}

	return *value
}

func parseRFC3339OrZero(value string) time.Time {
	if value == "" {
		return time.Time{}
	}

	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}
	}

	return parsed
}
