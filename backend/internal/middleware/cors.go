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

		// If no origin header, allow (same-origin request)
		if origin == "" {
			allowed = true
		} else {
			// Development mode always allows
			if cfg.Env == "development" {
				c.Header("Access-Control-Allow-Origin", origin)
				allowed = true
			} else {
				// Production: check explicit allowed origins first
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

				// Allow if no explicit domains configured and origin IP matches Host/IP
				// This handles IP-based access (e.g., http://192.168.0.211:2053)
				if !allowed && cfg.AllowedOrigins == "" && cfg.PanelDomain == "" {
					// Extract host from origin
					originHost := origin
					if strings.HasPrefix(originHost, "http://") {
						originHost = strings.TrimPrefix(originHost, "http://")
					} else if strings.HasPrefix(originHost, "https://") {
						originHost = strings.TrimPrefix(originHost, "https://")
					}
					originHost = strings.Split(originHost, ":")[0]

					// Get server host from headers
					serverHost := c.Request.Host
					serverHost = strings.Split(serverHost, ":")[0]

					// Allow if origin host matches server host
					if originHost == serverHost || originHost == "localhost" || originHost == "127.0.0.1" {
						c.Header("Access-Control-Allow-Origin", origin)
						allowed = true
					}
				}
			}
		}

		// Block if not allowed
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
