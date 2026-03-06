package copilot

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// TestIntegrationOAuthFlow verifies the complete device flow end-to-end using a mock server.
// It confirms that:
//   - client_id sent to the device code endpoint is the OpenCode value "Ov23li8tweQw6odWQebz"
//   - scope sent is "read:user" only (not "read:user user:email" or any other value)
//   - token exchange succeeds and returns a populated CopilotTokenData
func TestIntegrationOAuthFlow(t *testing.T) {
	const (
		expectedClientID    = "Ov23li8tweQw6odWQebz"
		expectedScope       = "read:user"
		mockDeviceCode      = "device-code-abc123"
		mockUserCode        = "ABCD-1234"
		mockVerificationURI = "https://github.com/login/device"
		mockAccessToken     = "ghu_integration_test_token"
	)

	var (
		capturedDeviceClientID string
		capturedDeviceScope    string
		capturedTokenClientID  string
	)

	// Mock server that handles both device code and token endpoints.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		switch r.URL.Path {
		case "/login/device/code":
			capturedDeviceClientID = r.FormValue("client_id")
			capturedDeviceScope = r.FormValue("scope")

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"device_code":      mockDeviceCode,
				"user_code":        mockUserCode,
				"verification_uri": mockVerificationURI,
				"expires_in":       900,
				"interval":         1,
			})

		case "/login/oauth/access_token":
			capturedTokenClientID = r.FormValue("client_id")

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": mockAccessToken,
				"token_type":   "bearer",
				"scope":        expectedScope,
			})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	client := &DeviceFlowClient{httpClient: newTestClient(srv)}

	// 1. Request device code and verify client_id / scope.
	deviceCode, err := client.RequestDeviceCode(context.Background())
	if err != nil {
		t.Fatalf("RequestDeviceCode: unexpected error: %v", err)
	}

	if capturedDeviceClientID != expectedClientID {
		t.Errorf("device code request client_id = %q, want %q", capturedDeviceClientID, expectedClientID)
	}

	if capturedDeviceScope != expectedScope {
		t.Errorf("device code request scope = %q, want %q (must be read:user only)", capturedDeviceScope, expectedScope)
	}

	if deviceCode.DeviceCode != mockDeviceCode {
		t.Errorf("DeviceCode = %q, want %q", deviceCode.DeviceCode, mockDeviceCode)
	}

	// 2. Simulate an immediate successful token exchange (interval=1 so it polls fast).
	deviceCode.Interval = 0 // Override to poll immediately without sleeping.

	tokenData, err := client.PollForToken(context.Background(), deviceCode)
	if err != nil {
		t.Fatalf("PollForToken: unexpected error: %v", err)
	}

	if capturedTokenClientID != expectedClientID {
		t.Errorf("token exchange request client_id = %q, want %q", capturedTokenClientID, expectedClientID)
	}

	if tokenData.AccessToken != mockAccessToken {
		t.Errorf("AccessToken = %q, want %q", tokenData.AccessToken, mockAccessToken)
	}

	if tokenData.Scope != expectedScope {
		t.Errorf("returned token scope = %q, want %q", tokenData.Scope, expectedScope)
	}
}

// TestIntegrationMockAuthConfig verifies that the package-level OAuth configuration
// constants reflect OpenCode identity values by calling RequestDeviceCode against a
// mock server and inspecting the posted form values.
func TestIntegrationMockAuthConfig(t *testing.T) {
	const (
		expectedClientID = "Ov23li8tweQw6odWQebz"
		expectedScope    = "read:user"
	)

	var capturedBody url.Values

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		capturedBody = r.Form

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"device_code":      "dc-test",
			"user_code":        "TEST-1234",
			"verification_uri": "https://github.com/login/device",
			"expires_in":       900,
			"interval":         5,
		})
	}))
	defer srv.Close()

	client := &DeviceFlowClient{httpClient: newTestClient(srv)}

	if _, err := client.RequestDeviceCode(context.Background()); err != nil {
		t.Fatalf("RequestDeviceCode: unexpected error: %v", err)
	}

	// Assert client_id matches OpenCode value, not the old VSCode value.
	gotClientID := capturedBody.Get("client_id")
	if gotClientID != expectedClientID {
		t.Errorf("client_id = %q, want %q (OpenCode client ID)", gotClientID, expectedClientID)
	}

	// Assert scope is read:user only — not the broader VSCode scope "read:user user:email".
	gotScope := capturedBody.Get("scope")
	if gotScope != expectedScope {
		t.Errorf("scope = %q, want %q (must not contain user:email)", gotScope, expectedScope)
	}

	if strings.Contains(gotScope, "user:email") {
		t.Errorf("scope %q must NOT contain 'user:email' — this is the old VSCode scope", gotScope)
	}
}

// TestIntegrationFullFlowHeaderVerification verifies that OAuth requests to GitHub APIs
// do not send VSCode-specific headers and use the Bearer authorization scheme.
func TestIntegrationFullFlowHeaderVerification(t *testing.T) {

	vscodeHeaders := []string{
		"Editor-Version",
		"Editor-Plugin-Version",
		"Copilot-Integration-Id",
		"X-Github-Api-Version",
		"X-Request-Id",
	}

	var capturedHeaders http.Header

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = r.Header.Clone()

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"login": "octocat",
			"email": "octocat@github.com",
			"name":  "The Octocat",
		})
	}))
	defer srv.Close()

	client := &DeviceFlowClient{httpClient: newTestClient(srv)}

	if _, err := client.FetchUserInfo(context.Background(), "test-token-abc"); err != nil {
		t.Fatalf("FetchUserInfo: unexpected error: %v", err)
	}

	// Verify that no VSCode-specific headers are included in OAuth requests.
	for _, header := range vscodeHeaders {
		if val := capturedHeaders.Get(header); val != "" {
			t.Errorf("VSCode header %q must NOT be present in OAuth requests, got %q", header, val)
		}
	}

	// Verify the Authorization header uses Bearer scheme (OpenCode style).
	authHeader := capturedHeaders.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		t.Errorf("Authorization = %q, want Bearer token scheme", authHeader)
	}
}
