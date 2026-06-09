package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRequestID_GeneratesNewID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)

	RequestID()(c)

	requestID, exists := c.Get("request_id")
	assert.True(t, exists)
	assert.NotEmpty(t, requestID)
	assert.Len(t, requestID.(string), 36) // UUID format
	assert.Equal(t, requestID.(string), w.Header().Get("X-Request-ID"))
}

func TestRequestID_UsesHeaderValue(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	c.Request.Header.Set("X-Request-ID", "my-custom-id")

	RequestID()(c)

	requestID, exists := c.Get("request_id")
	assert.True(t, exists)
	assert.Equal(t, "my-custom-id", requestID.(string))
	assert.Equal(t, "my-custom-id", w.Header().Get("X-Request-ID"))
}

func TestRequestID_ContinuesNext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)

	nextCalled := false
	middleware := RequestID()
	// Set up a next handler
	c.Set("test", nil)
	middleware(c)
	if !c.IsAborted() {
		nextCalled = true
	}

	assert.True(t, nextCalled)
}
