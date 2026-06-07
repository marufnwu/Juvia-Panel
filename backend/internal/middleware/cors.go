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
		allowed := false

		// Get host from X-Forwarded-Host header (set by reverse proxy) or Host header
		host := c.Request.Header.Get("X-Forwarded-Host")
		if host == "" {
			host = c.Request.Host
		}

		// Same-IP check: extract IP/hostname from origin and compare with host (ignoring ports)
		if origin != "" {
			originHost := origin
			if strings.HasPrefix(origin, "http://") {
				originHost = strings.TrimPrefix(origin, "http://")
			} else if strings.HasPrefix(origin, "https://") {
				originHost = strings.TrimPrefix(origin, "https://")
			}
			// Remove port from origin host
			if strings.Contains(originHost, ":") {
				originHost = strings.Split(originHost, ":")[0]
			}
			// Remove port from server host
			hostOnly := host
			if strings.Contains(hostOnly, ":") {
				hostOnly = strings.Split(hostOnly, ":")[0]
			}
			// Compare IPs or hostnames
			if originHost == hostOnly || originHost == "localhost" || originHost == "127.0.0.1" {
				c.Header("Access-Control-Allow-Origin", origin)
				allowed = true
			}
		}

		if cfg.Env == "development" {
			c.Header("Access-Control-Allow-Origin", origin)
			allowed = true
		} else if origin != "" && !allowed {
			// In production, check AllowedOrigins
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
