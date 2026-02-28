package auth

import (
	"context"
	"fmt"
	"time"

	kiloauth "github.com/router-for-me/CLIProxyAPI/v6/internal/auth/kilo"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/config"
	coreauth "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/auth"
)

// KiloAuthenticator implements the login flow for Kilo AI accounts.
type KiloAuthenticator struct{}

// NewKiloAuthenticator constructs a Kilo authenticator.
func NewKiloAuthenticator() *KiloAuthenticator {
	return &KiloAuthenticator{}
}

// Provider returns the provider identifier for Kilo.
func (a *KiloAuthenticator) Provider() string {
	return "kilo"
}

// RefreshLead returns nil since Kilo does not require proactive token refresh.
func (a *KiloAuthenticator) RefreshLead() *time.Duration {
	return nil
}

// Login manages the device flow authentication for Kilo AI.
// It initiates a device code flow, polls for token confirmation, fetches the user
// profile and org info, then returns a persisted auth record.
//
// Parameters:
//   - ctx: The context for cancellation and deadlines.
//   - cfg: The application configuration (must not be nil).
//   - opts: Login options including interactive prompt callback.
//
// Returns:
//   - *coreauth.Auth: The resulting auth record on success.
//   - error: Non-nil if any step of the flow fails.
func (a *KiloAuthenticator) Login(ctx context.Context, cfg *config.Config, opts *LoginOptions) (*coreauth.Auth, error) {
	if cfg == nil {
		return nil, fmt.Errorf("cliproxy auth: configuration is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if opts == nil {
		opts = &LoginOptions{}
	}

	// 1. Initiate the device authorization flow — get verification URL and code.
	kilocodeAuth := kiloauth.NewKiloAuth()

	fmt.Println("Initiating Kilo device authentication...")
	resp, err := kilocodeAuth.InitiateDeviceFlow(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to initiate device flow: %w", err)
	}

	fmt.Printf("Please visit: %s\n", resp.VerificationURL)
	fmt.Printf("And enter code: %s\n", resp.Code)

	// 2. Poll until the user completes browser authorization.
	fmt.Println("Waiting for authorization...")
	status, err := kilocodeAuth.PollForToken(ctx, resp.Code)
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	fmt.Printf("Authentication successful for %s\n", status.UserEmail)

	// 3. Fetch the user profile to discover organization memberships.
	profile, err := kilocodeAuth.GetProfile(ctx, status.Token)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch profile: %w", err)
	}

	// 4. Select the organization interactively if multiple are available.
	var orgID string
	if len(profile.Orgs) > 1 {
		fmt.Println("Multiple organizations found. Please select one:")
		for i, org := range profile.Orgs {
			fmt.Printf("[%d] %s (%s)\n", i+1, org.Name, org.ID)
		}

		if opts.Prompt != nil {
			input, errPrompt := opts.Prompt("Enter the number of the organization: ")
			if errPrompt != nil {
				return nil, errPrompt
			}
			var choice int
			_, errScan := fmt.Sscan(input, &choice)
			if errScan == nil && choice > 0 && choice <= len(profile.Orgs) {
				orgID = profile.Orgs[choice-1].ID
			} else {
				orgID = profile.Orgs[0].ID
				fmt.Printf("Invalid choice, defaulting to %s\n", profile.Orgs[0].Name)
			}
		} else {
			orgID = profile.Orgs[0].ID
			fmt.Printf("Non-interactive mode, defaulting to organization: %s\n", profile.Orgs[0].Name)
		}
	} else if len(profile.Orgs) == 1 {
		orgID = profile.Orgs[0].ID
	}

	// 5. Fetch defaults (preferred model, etc.) for the selected organization.
	defaults, err := kilocodeAuth.GetDefaults(ctx, status.Token, orgID)
	if err != nil {
		fmt.Printf("Warning: failed to fetch defaults: %v\n", err)
		defaults = &kiloauth.Defaults{}
	}

	ts := &kiloauth.KiloTokenStorage{
		Token:          status.Token,
		OrganizationID: orgID,
		Model:          defaults.Model,
		Email:          status.UserEmail,
		Type:           "kilo",
	}

	fileName := kiloauth.CredentialFileName(status.UserEmail)
	metadata := map[string]any{
		"email":           status.UserEmail,
		"organization_id": orgID,
		"model":           defaults.Model,
	}

	return &coreauth.Auth{
		ID:       fileName,
		Provider: a.Provider(),
		FileName: fileName,
		Storage:  ts,
		Metadata: metadata,
	}, nil
}
