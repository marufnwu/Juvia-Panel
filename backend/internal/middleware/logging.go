package middleware

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

// Logging middleware logs HTTP requests.
func Logging() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		start := time.Now()

		// Process request
		c.Next()

		// Log after request
		duration := time.Since(start)
		log.Printf(
			"[%s] %s %s %d %s",
			c.GetString("request_id"),
			c.Request.Method,
			c.Request.URL.Path,
			c.Writer.Status(),
			duration,
		)
	}
}
