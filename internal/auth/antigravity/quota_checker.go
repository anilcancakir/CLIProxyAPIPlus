package antigravity

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/tidwall/gjson"
)

// ModelQuota represents the quota status for a specific model.
type ModelQuota struct {
	// Name is the unique identifier for the model (e.g., "gemini-1.5-pro").
	Name string `json:"name"`
	// RemainingFraction is the fraction of remaining quota (0.0 to 1.0).
	RemainingFraction float64 `json:"remaining_fraction"`
	// ResetTime is when the quota resets.
	ResetTime time.Time `json:"reset_time"`
	// DisplayName is the human-readable name of the model.
	DisplayName string `json:"display_name"`
	// RemainingPercent is the percentage of remaining quota (0 to 100).
	// This is computed from RemainingFraction.
	RemainingPercent float64 `json:"remaining_percent"`
}

// QuotaData represents the collection of model quotas for an account.
type QuotaData struct {
	// Models is the list of model-specific quotas.
	Models []ModelQuota `json:"models"`
	// LastUpdated is the timestamp of the last quota fetch.
	LastUpdated time.Time `json:"last_updated"`
	// IsForbidden indicates whether access to the project is forbidden.
	IsForbidden bool `json:"is_forbidden"`
}

// StoreQuota directly stores quota data in the cache for the given accountID.
func (c *AntigravityQuotaChecker) StoreQuota(accountID string, data *QuotaData) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache[accountID] = data
}

/**
 * AddModel appends a ModelQuota to the QuotaData's Models slice.
 *
 * @param  ModelQuota  $model  The model quota to add.
 * @return void
 */
func (q *QuotaData) AddModel(model ModelQuota) {
	q.Models = append(
		q.Models,
		model,
	)
}

// QuotaChecker defines the contract for fetching Antigravity quota information.
type QuotaChecker interface {
	/**
	 * FetchQuota retrieves quota information for the specified account.
	 *
	 * @param  context.Context  $ctx          The execution context.
	 * @param  string           $accessToken  The OAuth access token.
	 * @param  string           $email        The account email address.
	 * @param  string           $accountID    The account unique identifier.
	 * @return *QuotaData
	 *
	 * @throws error
	 */
	FetchQuota(
		ctx context.Context,
		accessToken string,
		email string,
		accountID string,
	) (*QuotaData, error)
}

// AntigravityQuotaChecker implements QuotaChecker using the fetchAvailableModels API.
type AntigravityQuotaChecker struct {
	mu         sync.RWMutex
	cache      map[string]*QuotaData
	httpClient *http.Client
	baseURL    string
}

/**
 * NewAntigravityQuotaChecker creates a new checker with the given HTTP client.
 *
 * @param  *http.Client  $httpClient  The HTTP client to use.
 * @return *AntigravityQuotaChecker
 */
func NewAntigravityQuotaChecker(httpClient *http.Client) *AntigravityQuotaChecker {
	return &AntigravityQuotaChecker{
		cache:      make(map[string]*QuotaData),
		httpClient: httpClient,
		baseURL:    "https://cloudcode-pa.googleapis.com",
	}
}

/**
 * GetCachedQuota returns the most recently fetched quota for the account.
 *
 * @param  string  $accountID  The account unique identifier.
 * @return *QuotaData|nil
 */
func (c *AntigravityQuotaChecker) GetCachedQuota(accountID string) *QuotaData {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.cache[accountID]
}

/**
 * FetchQuota retrieves quota information for the specified account.
 *
 * @param  context.Context  $ctx          The execution context.
 * @param  string           $accessToken  The OAuth access token.
 * @param  string           $email        The account email address.
 * @param  string           $accountID    The account unique identifier.
 * @return *QuotaData
 *
 * @throws error
 */
func (c *AntigravityQuotaChecker) FetchQuota(
	ctx context.Context,
	accessToken string,
	email string,
	accountID string,
) (*QuotaData, error) {
	url := c.baseURL + "/v1internal:fetchAvailableModels"

	var resp *http.Response
	var err error

	// 1. Retry loop for network errors (max 3 attempts).
	for i := 0; i < 3; i++ {
		req, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader([]byte("{}")))
		if reqErr != nil {
			return nil, fmt.Errorf("failed to create request: %w", reqErr)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+accessToken)
		req.Header.Set("User-Agent", "google-api-nodejs-client/9.15.1 gl-node/22.17.0")

		resp, err = c.httpClient.Do(req)
		if err == nil {
			break
		}

		if i < 2 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(1 * time.Second):
				continue
			}
		}
	}

	if err != nil {
		return nil, fmt.Errorf("quota fetch failed after retries: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	// 2. Handle HTTP status codes.
	if resp.StatusCode == http.StatusForbidden {
		quotaData := &QuotaData{
			Models:      []ModelQuota{},
			LastUpdated: time.Now(),
			IsForbidden: true,
		}
		c.mu.Lock()
		c.cache[accountID] = quotaData
		c.mu.Unlock()
		return quotaData, nil
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("rate limited fetching quota")
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("quota fetch failed: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// 3. Parse JSON response and filter models.
	result := gjson.GetBytes(body, "models")
	if !result.Exists() {
		return nil, fmt.Errorf("invalid response: missing models field")
	}

	quotaData := &QuotaData{
		Models:      make([]ModelQuota, 0),
		LastUpdated: time.Now(),
		IsForbidden: false,
	}

	for key, value := range result.Map() {
		name := strings.TrimSpace(key)
		if name == "" {
			continue
		}

		// Filter specific model prefixes
		if !strings.HasPrefix(name, "gemini") &&
			!strings.HasPrefix(name, "claude") &&
			!strings.HasPrefix(name, "gpt") &&
			!strings.HasPrefix(name, "image") &&
			!strings.HasPrefix(name, "imagen") {
			continue
		}

		displayName := value.Get("displayName").String()
		if displayName == "" {
			displayName = name
		}

		remainingFraction := value.Get("quotaInfo.remainingFraction").Float()
		resetTimeStr := value.Get("quotaInfo.resetTime").String()

		var resetTime time.Time
		if resetTimeStr != "" {
			parsedTime, tErr := time.Parse(time.RFC3339, resetTimeStr)
			if tErr == nil {
				resetTime = parsedTime
			}
		}

		modelQuota := ModelQuota{
			Name:              name,
			DisplayName:       displayName,
			RemainingFraction: remainingFraction,
			RemainingPercent:  remainingFraction * 100,
			ResetTime:         resetTime,
		}
		quotaData.AddModel(modelQuota)
	}

	// 4. Update cache and return.
	c.mu.Lock()
	c.cache[accountID] = quotaData
	c.mu.Unlock()

	return quotaData, nil
}
