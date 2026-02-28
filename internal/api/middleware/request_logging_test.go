package middleware

// NOTE: The fork (kooshapari) added shouldSkipMethodForRequestLogging and
// shouldCaptureRequestBody helpers that do not exist in this origin repo.
// The tests below are adapted to test shouldLogRequest which DOES exist,
// covering the path-based filtering logic equivalent.

import (
	"testing"
)

func TestShouldLogRequest_ManagementPathsSkipped(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "management v0 path should not log",
			path: "/v0/management/tokens",
			want: false,
		},
		{
			name: "management path should not log",
			path: "/management/status",
			want: false,
		},
		{
			name: "v1 chat completions should log",
			path: "/v1/chat/completions",
			want: true,
		},
		{
			name: "v1 responses should log",
			path: "/v1/responses",
			want: true,
		},
		{
			name: "api provider path should log",
			path: "/api/provider/list",
			want: true,
		},
		{
			name: "api non-provider path should not log",
			path: "/api/internal/debug",
			want: false,
		},
		{
			name: "root path should log",
			path: "/",
			want: true,
		},
	}

	for _, tc := range tests {
		got := shouldLogRequest(tc.path)
		if got != tc.want {
			t.Fatalf("%s: shouldLogRequest(%q) = %t, want %t", tc.name, tc.path, got, tc.want)
		}
	}
}
