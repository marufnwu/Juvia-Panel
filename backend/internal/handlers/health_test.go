package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"panel-api/internal/config"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestHealthCheck_ReturnsOK(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		Version: "1.0.0",
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/health", nil)
	c.Set("config", cfg)
	c.Set("request_id", "req-123")

	HealthCheck(c)

	assert.Equal(t, http.StatusOK, w.Code)

	body := w.Body.String()
	assert.Contains(t, body, `"status":"ok"`)
	assert.Contains(t, body, `"version":"1.0.0"`)
	assert.Contains(t, body, `"request_id":"req-123"`)
}

func TestHealthCheck_UnknownVersion(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/health", nil)
	c.Set("request_id", "req-456")

	// No config set
	HealthCheck(c)

	assert.Equal(t, http.StatusOK, w.Code)

	body := w.Body.String()
	assert.Contains(t, body, `"version":"unknown"`)
}

func TestHealthCheck_ReturnsTimestamp(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/health", nil)
	c.Set("config", cfg)

	before := time.Now().UTC()
	HealthCheck(c)
	after := time.Now().UTC()

	assert.Equal(t, http.StatusOK, w.Code)

	body := w.Body.String()
	assert.Contains(t, body, `"time":`)
	assert.Contains(t, body, before.Format("2006-01-02T"))
	_ = after
}
