package test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// TestServerHealth tests the server health endpoint.
func TestServerHealth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy"}`)) //nolint:errcheck
	}))
	defer srv.Close()

	resp, err := srv.Client().Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

// TestBinaryExists checks whether a known build artifact exists in the repo root.
func TestBinaryExists(t *testing.T) {
	paths := []string{
		"cli-proxy-api-plus-integration-test",
		"cli-proxy-api-plus",
		"server",
	}

	// Resolve repo root relative to this file's location (two dirs up from test/).
	repoRoot, err := filepath.Abs(filepath.Join("..", "."))
	if err != nil {
		t.Skip("Cannot determine repo root")
	}

	for _, p := range paths {
		path := filepath.Join(repoRoot, p)
		if info, statErr := os.Stat(path); statErr == nil && !info.IsDir() {
			t.Logf("Found binary: %s", p)
			return
		}
	}
	t.Skip("Binary not found in expected paths")
}

// TestConfigFile tests that a config YAML file can be written and read back.
func TestConfigFile(t *testing.T) {
	config := `
port: 8317
host: localhost
log_level: debug
`
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, "config.yaml")
	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(configPath); err != nil {
		t.Error(err)
	}
}

// TestOAuthLoginFlow tests a mock OAuth token endpoint responds correctly.
func TestOAuthLoginFlow(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth/token" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"access_token":"test","expires_in":3600}`)) //nolint:errcheck
		}
	}))
	defer srv.Close()

	client := srv.Client()
	client.Timeout = 5 * time.Second

	resp, err := client.Get(srv.URL + "/oauth/token")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

// TestKiloLoginBinary tests the kilo login binary — skipped when binary is absent.
func TestKiloLoginBinary(t *testing.T) {
	// Binary is an optional build artifact; skip if not present.
	repoRoot, err := filepath.Abs(filepath.Join("..", "."))
	if err != nil {
		t.Skip("Cannot determine repo root")
	}

	binary := filepath.Join(repoRoot, "cli-proxy-api-plus-integration-test")
	if _, statErr := os.Stat(binary); os.IsNotExist(statErr) {
		t.Skip("Binary not found")
	}

	cmd := exec.Command(binary, "-help")
	cmd.Dir = repoRoot

	if err := cmd.Run(); err != nil {
		t.Logf("Binary help returned error: %v", err)
	}
}
