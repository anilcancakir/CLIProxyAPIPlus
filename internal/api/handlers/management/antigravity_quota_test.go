package management

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestGetAntigravityQuotaNoHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/v0/management/antigravity-quota", nil)

	// nil handler — must not panic, returns 200 with empty array
	var h *Handler
	h.GetAntigravityQuota(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var result []interface{}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
	assert.Empty(t, result)
}

func TestGetAntigravityQuotaNoAuthManager(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/v0/management/antigravity-quota", nil)

	// Handler with nil authManager → returns [] with 200.
	h := &Handler{}
	h.GetAntigravityQuota(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var result []interface{}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
	assert.Empty(t, result)
}
