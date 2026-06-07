package middleware

import (
	"panel-api/internal/config"
	"strings"

	"github.com/gin-gonic/gin"
)

func CORS(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		allowed := false

		if origin == "" {
			allowed = true
		} else if cfg.Env == "development" {
			c.Header("Access-Control-Allow-Origin", origin)
			allowed = true
		} else if cfg.AllowedOrigins != "" {
			allowedList := strings.Split(cfg.AllowedOrigins, ",")
			for _, allowedOrigin := range allowedList {
				if strings.TrimSpace(allowedOrigin) == origin {
					c.Header("Access-Control-Allow-Origin", origin)
					allowed = true
					break
				}
			}
		} else if cfg.PanelDomain != "" && strings.Contains(origin, cfg.PanelDomain) {
			c.Header("Access-Control-Allow-Origin", origin)
			allowed = true
		} else if cfg.AllowedOrigins == "" && cfg.PanelDomain == "" {
			c.Header("Access-Control-Allow-Origin", origin)
			allowed = true
		}

		if !allowed {
			c.AbortWithStatusJSON(403, gin.H{"error": "origin not allowed"})
			return
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Request-ID")
		c.Header("Access-Control-Expose-Headers", "X-Request-ID")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}