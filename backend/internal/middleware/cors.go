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
			if cfg.Env == "production" {
				c.AbortWithStatusJSON(403, gin.H{"error": "origin not allowed"})
				return
			}
			c.Header("Access-Control-Allow-Origin", origin)
			allowed = true
		} else if isLocalOrigin(origin) {
			// Allow localhost and IP address access for local network
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

// isLocalOrigin checks if the origin is localhost or an IP address
func isLocalOrigin(origin string) bool {
	// Remove protocol
	origin = strings.TrimPrefix(origin, "http://")
	origin = strings.TrimPrefix(origin, "https://")
	
	// Check for localhost
	if strings.HasPrefix(origin, "localhost") {
		return true
	}
	
	// Check for IP address (simple check for IPv4)
	parts := strings.Split(origin, ":")
	host := parts[0]
	
	// Check if it's an IP address (4 parts separated by dots)
	ipParts := strings.Split(host, ".")
	if len(ipParts) == 4 {
		// Simple validation: each part should be a number 0-255
		for _, part := range ipParts {
			if part == "" {
				return false
			}
			for _, c := range part {
				if c < '0' || c > '9' {
					return false
				}
			}
		}
		return true
	}
	
	return false
}