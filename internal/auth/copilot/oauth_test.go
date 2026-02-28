package copilot

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// roundTripFunc lets us inject a custom transport for testing.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// newTestClient returns an *http.Client whose requests are redirected to the given test server,
// regardless of the original URL host.
func newTestClient(srv *httptest.Server) *http.Client {
	return &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			req2 := req.Clone(req.Context())
			req2.URL.Scheme = "http"
			req2.URL.Host = strings.TrimPrefix(srv.URL, "http://")
			return srv.Client().Transport.RoundTrip(req2)
		}),
	}
}

// TestFetchUserInfo_ReturnsLogin verifies that FetchUserInfo returns the login string.
// Origin's FetchUserInfo returns (string, error) — only the login field.
func TestFetchUserInfo_ReturnsLogin(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"login": "octocat",
			"email": "octocat@github.com",
			"name":  "The Octocat",
		})
	}))
	defer srv.Close()

	client := &DeviceFlowClient{httpClient: newTestClient(srv)}
	login, err := client.FetchUserInfo(context.Background(), "test-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if login.Login != "octocat" {
		t.Errorf("Login: got %q, want %q", login.Login, "octocat")
	}

}

// TestFetchUserInfo_EmptyToken verifies error is returned for empty access token.
func TestFetchUserInfo_EmptyToken(t *testing.T) {
	client := &DeviceFlowClient{httpClient: http.DefaultClient}
	_, err := client.FetchUserInfo(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty token, got nil")
	}
}

// TestFetchUserInfo_EmptyLogin verifies error is returned when API returns no login.
func TestFetchUserInfo_EmptyLogin(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"email":"someone@example.com","name":"No Login"}`))
	}))
	defer srv.Close()

	client := &DeviceFlowClient{httpClient: newTestClient(srv)}
	_, err := client.FetchUserInfo(context.Background(), "test-token")
	if err == nil {
		t.Fatal("expected error for empty login, got nil")
	}
}

// TestFetchUserInfo_HTTPError verifies error is returned on non-2xx response.
func TestFetchUserInfo_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"Bad credentials"}`))
	}))
	defer srv.Close()

	client := &DeviceFlowClient{httpClient: newTestClient(srv)}
	_, err := client.FetchUserInfo(context.Background(), "bad-token")
	if err == nil {
		t.Fatal("expected error for 401 response, got nil")
	}
}

// TestCopilotTokenStorage_BasicFields verifies token storage fields serialise correctly.
func TestCopilotTokenStorage_BasicFields(t *testing.T) {
	ts := &CopilotTokenStorage{
		AccessToken: "ghu_abc",
		TokenType:   "bearer",
		Scope:       "read:user user:email",
		Username:    "octocat",
		Type:        "github-copilot",
	}

	data, err := json.Marshal(ts)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var out map[string]any
	if err = json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	for _, key := range []string{"access_token", "username", "type"} {
		if _, ok := out[key]; !ok {
			t.Errorf("expected key %q in JSON output, not found", key)
		}
	}
	if out["access_token"] != "ghu_abc" {
		t.Errorf("access_token: got %v, want %q", out["access_token"], "ghu_abc")
	}
}

// TestCopilotAuthBundle_Fields verifies bundle carries username through the pipeline.
func TestCopilotAuthBundle_Fields(t *testing.T) {
	bundle := &CopilotAuthBundle{
		TokenData: &CopilotTokenData{AccessToken: "ghu_abc"},
		Username:  "octocat",
	}
	if bundle.Username != "octocat" {
		t.Errorf("bundle.Username: got %q, want %q", bundle.Username, "octocat")
	}
	if bundle.TokenData == nil || bundle.TokenData.AccessToken != "ghu_abc" {
		t.Errorf("bundle.TokenData.AccessToken: got unexpected value")
	}
}
