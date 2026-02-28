package antigravity

import (
	"context"
	"time"
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
}
