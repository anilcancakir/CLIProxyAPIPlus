package middleware

// NOTE: The fork added requestBodyOverrideContextKey and extractRequestBody
// that do not exist in this origin repo. Tests below are adapted to test
// ResponseWriterWrapper and RequestInfo construction which DO exist.

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestNewResponseWriterWrapper_InitializesCorrectly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	info := &RequestInfo{Body: []byte("test-body")}
	wrapper := NewResponseWriterWrapper(c.Writer, nil, info)

	if wrapper == nil {
		t.Fatal("expected non-nil ResponseWriterWrapper")
	}
	if wrapper.requestInfo == nil {
		t.Fatal("expected non-nil requestInfo")
	}
	if string(wrapper.requestInfo.Body) != "test-body" {
		t.Fatalf("requestInfo.Body: got %q, want %q", string(wrapper.requestInfo.Body), "test-body")
	}
}

func TestResponseWriterWrapper_ShouldBufferResponseBody_LogOnErrorOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	wrapper := NewResponseWriterWrapper(c.Writer, nil, &RequestInfo{})
	wrapper.logOnErrorOnly = true

	// With status code 0 (not yet set) and logOnErrorOnly, shouldBufferResponseBody depends on status.
	// The default recorder returns 200 via Status() — which is < 400, so should NOT buffer.
	if wrapper.shouldBufferResponseBody() {
		t.Fatal("expected shouldBufferResponseBody to be false for 2xx status with logOnErrorOnly")
	}

	// Simulate an error status code
	wrapper.statusCode = 500
	if !wrapper.shouldBufferResponseBody() {
		t.Fatal("expected shouldBufferResponseBody to be true for 5xx status with logOnErrorOnly")
	}
}
