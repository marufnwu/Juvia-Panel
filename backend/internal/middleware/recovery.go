package middleware

import (
	"log"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

// Recovery middleware recovers from panics.
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("[%s] Panic recovered: %v\n%s", c.GetString("request_id"), err, debug.Stack())
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error":      "internal_server_error",
					"message":    "An unexpected error occurred",
					"request_id": c.GetString("request_id"),
				})
			}
		}()
		c.Next()
	}
}
