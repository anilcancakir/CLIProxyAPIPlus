package antigravity

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/tidwall/gjson"
)

/**
 * ModelQuota represents the quota status for a specific model.
 */
type ModelQuota struct {
	// Name is the unique identifier for the model (e.g., "gemini-1.5-pro").
	Name string `json:"name"`

	// RemainingFraction is the fraction of remaining quota (0.0 to 1.0).
	RemainingFraction float64 `json:"remaining_fraction"`

	// ResetTime is when the quota resets.
	ResetTime time.Time `json:"reset_time"`

	// DisplayName is the human-readable name of the model.
	DisplayName string `json:"display_name"`
}

/**
 * RemainingPercent returns the percentage of remaining quota (0 to 100).
 *
 * @return float64
 */
func (m ModelQuota) RemainingPercent() float64 {
	return m.RemainingFraction * 100
}

/**
 * QuotaData represents the collection of model quotas for an account.
 */
type QuotaData struct {
	// Models is the list of model-specific quotas.
	Models []ModelQuota `json:"models"`

	// LastUpdated is the timestamp of the last quota fetch.
	LastUpdated time.Time `json:"last_updated"`

	// IsForbidden indicates whether access to the project is forbidden.
	IsForbidden bool `json:"is_forbidden"`
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

/**
 * QuotaChecker defines the contract for fetching Antigravity quota information.
 */
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

	/**
	 * GetCachedQuota returns the most recently fetched quota for the account.
	 *
	 * @param  string  $accountID  The account unique identifier.
	 * @return *QuotaData|nil
	 */
	GetCachedQuota(accountID string) *QuotaData

	/**
	 * StoreQuota saves a fetched quota directly into the cache.
	 *
	 * @param  string      $accountID  The account unique identifier.
	 * @param  *QuotaData  $data       The quota data to store.
	 * @return void
	 */
	StoreQuota(accountID string, data *QuotaData)
}

/**
 * AntigravityQuotaChecker implements QuotaChecker using the fetchAvailableModels API.
 */
type AntigravityQuotaChecker struct {
	httpClient *http.Client
	mu         sync.RWMutex
	cache      map[string]*QuotaData
}

/**
 * NewAntigravityQuotaChecker creates a new checker with the given HTTP client.
 *
 * @param  *http.Client  $httpClient  The HTTP client to use for requests.
 * @return *AntigravityQuotaChecker
 */
func NewAntigravityQuotaChecker(httpClient *http.Client) *AntigravityQuotaChecker {
	return &AntigravityQuotaChecker{
		httpClient: httpClient,
		cache:      make(map[string]*QuotaData),
	}
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
func (q *AntigravityQuotaChecker) FetchQuota(
	ctx context.Context,
	accessToken string,
	email string,
	accountID string,
) (*QuotaData, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		"https://cloudcode-pa.googleapis.com/v1internal:fetchAvailableModels",
		strings.NewReader("{}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := q.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden {
		data := &QuotaData{
			IsForbidden: true,
			LastUpdated: time.Now(),
		}
		q.mu.Lock()
		q.cache[accountID] = data
		q.mu.Unlock()
		return data, nil
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("rate limited fetching quota for %s", accountID)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	data := &QuotaData{
		LastUpdated: time.Now(),
	}

	result := gjson.GetBytes(bodyBytes, "models")
	result.ForEach(func(key, value gjson.Result) bool {
		name := key.String()
		// Only keep relevant models
		if !strings.HasPrefix(name, "gemini") &&
			!strings.HasPrefix(name, "claude") &&
			!strings.HasPrefix(name, "gpt") &&
			!strings.HasPrefix(name, "image") &&
			!strings.HasPrefix(name, "imagen") {
			return true // continue
		}

		quotaInfo := value.Get("quotaInfo")
		if !quotaInfo.Exists() {
			return true // continue
		}

		remainingFraction := quotaInfo.Get("remainingFraction").Float()
		resetTimeString := quotaInfo.Get("resetTime").String()

		var resetTime time.Time
		if resetTimeString != "" {
			parsedTime, parseErr := time.Parse(time.RFC3339, resetTimeString)
			if parseErr == nil {
				resetTime = parsedTime
			}
		}

		data.AddModel(ModelQuota{
			Name:              name,
			RemainingFraction: remainingFraction,
			ResetTime:         resetTime,
			DisplayName:       name, // Use name as display name for now
		})

		return true // continue
	})

	q.mu.Lock()
	q.cache[accountID] = data
	q.mu.Unlock()

	return data, nil
}

/**
 * GetCachedQuota returns the most recently fetched quota for the account.
 *
 * @param  string  $accountID  The account unique identifier.
 * @return *QuotaData|nil
 */
func (q *AntigravityQuotaChecker) GetCachedQuota(accountID string) *QuotaData {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.cache[accountID]
}

/**
 * StoreQuota saves a fetched quota directly into the cache.
 *
 * @param  string      $accountID  The account unique identifier.
 * @param  *QuotaData  $data       The quota data to store.
 * @return void
 */
func (q *AntigravityQuotaChecker) StoreQuota(accountID string, data *QuotaData) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.cache[accountID] = data
}
