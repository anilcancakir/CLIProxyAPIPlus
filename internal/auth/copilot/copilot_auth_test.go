package copilot

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetCopilotAPIToken_PassThrough(t *testing.T) {
	t.Parallel()

	auth := &CopilotAuth{}
	apiToken, err := auth.GetCopilotAPIToken(context.Background(), "ghu_test_token")
	if err != nil {
		t.Fatalf("GetCopilotAPIToken() error = %v", err)
	}
	if apiToken == nil {
		t.Fatal("GetCopilotAPIToken() returned nil token")
	}
	if apiToken.Token != "ghu_test_token" {
		t.Fatalf("token = %q, want %q", apiToken.Token, "ghu_test_token")
	}
	if apiToken.ExpiresAt != 0 {
		t.Fatalf("expires_at = %d, want 0", apiToken.ExpiresAt)
	}
	if apiToken.Endpoints.API != "" {
		t.Fatalf("endpoints.api = %q, want empty", apiToken.Endpoints.API)
	}
}

func TestGetCopilotAPIToken_EmptyToken(t *testing.T) {
	t.Parallel()

	auth := &CopilotAuth{}
	_, err := auth.GetCopilotAPIToken(context.Background(), "")
	if err == nil {
		t.Fatal("GetCopilotAPIToken() error = nil, want error")
	}
}

func TestLoadAndValidateToken_UsesGitHubUserInfoEndpoint(t *testing.T) {
	t.Parallel()

	var capturedAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"login": "octocat",
		})
	}))
	defer srv.Close()

	client := newTestClient(srv)
	auth := &CopilotAuth{
		httpClient: client,
		deviceClient: &DeviceFlowClient{
			httpClient: client,
		},
	}

	ok, err := auth.LoadAndValidateToken(context.Background(), &CopilotTokenStorage{AccessToken: "ghu_valid"})
	if err != nil {
		t.Fatalf("LoadAndValidateToken() error = %v", err)
	}
	if !ok {
		t.Fatal("LoadAndValidateToken() = false, want true")
	}
	if capturedAuth != "Bearer ghu_valid" {
		t.Fatalf("Authorization = %q, want %q", capturedAuth, "Bearer ghu_valid")
	}
}

func TestMakeAuthenticatedRequest_UsesOpenCodeHeaders(t *testing.T) {
	t.Parallel()

	auth := &CopilotAuth{}
	request, err := auth.MakeAuthenticatedRequest(
		context.Background(),
		http.MethodPost,
		"https://api.githubcopilot.com/models",
		nil,
		&CopilotAPIToken{Token: "ghu_token"},
	)
	if err != nil {
		t.Fatalf("MakeAuthenticatedRequest() error = %v", err)
	}

	if got := request.Header.Get("Authorization"); got != "Bearer ghu_token" {
		t.Fatalf("Authorization = %q, want %q", got, "Bearer ghu_token")
	}
	if got := request.Header.Get("User-Agent"); got != "opencode/1.2.27" {
		t.Fatalf("User-Agent = %q, want %q", got, "opencode/1.2.27")
	}
	if got := request.Header.Get("Openai-Intent"); got != "conversation-edits" {
		t.Fatalf("Openai-Intent = %q, want %q", got, "conversation-edits")
	}
	if got := request.Header.Get("X-Initiator"); got != "user" {
		t.Fatalf("X-Initiator = %q, want %q", got, "user")
	}

	for _, headerName := range []string{
		"Editor-Version",
		"Editor-Plugin-Version",
		"Copilot-Integration-Id",
	} {
		if got := request.Header.Get(headerName); got != "" {
			t.Fatalf("%s = %q, want empty", headerName, got)
		}
	}
}
