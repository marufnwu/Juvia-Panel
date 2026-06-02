package middleware

import (
	"panel-api/internal/config"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORS middleware handles Cross-Origin Resource Sharing.
func CORS(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		host := c.Request.Host
		allowed := false

		// Same-host check: compare IP/hostname from Origin with Host header
		if origin != "" {
			originHost := strings.Split(origin, "://")[1]
			if strings.Contains(originHost, ":") {
				originHost = strings.Split(originHost, ":")[0]
			}
			hostOnly := strings.Split(host, ":")[0]
			if originHost == hostOnly {
				c.Header("Access-Control-Allow-Origin", origin)
				allowed = true
			}
		}

		if cfg.Env == "development" {
			c.Header("Access-Control-Allow-Origin", origin)
			allowed = true
		} else if origin != "" && !allowed {
			// In production, check allowed origins
			if cfg.AllowedOrigins != "" {
				allowedList := strings.Split(cfg.AllowedOrigins, ",")
				for _, allowedOrigin := range allowedList {
					if strings.TrimSpace(allowedOrigin) == origin {
						c.Header("Access-Control-Allow-Origin", origin)
						allowed = true
						break
					}
				}
			}

			// Allow if origin contains PanelDomain
			if !allowed && cfg.PanelDomain != "" && strings.Contains(origin, cfg.PanelDomain) {
				c.Header("Access-Control-Allow-Origin", origin)
				allowed = true
			}
		}

		// Block cross-origin requests in production if not allowed
		if cfg.Env == "production" && origin != "" && !allowed {
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
