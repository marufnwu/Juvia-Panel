package middleware

import (
	"panel-api/internal/config"
	"strings"

	"github.com/gin-gonic/gin"
)

func CORS(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		host := c.Request.Host
		allowed := false

		if origin == "" {
			allowed = true
		} else {
			if cfg.Env == "development" {
				c.Header("Access-Control-Allow-Origin", origin)
				allowed = true
			} else {
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

				if !allowed && cfg.PanelDomain != "" && strings.Contains(origin, cfg.PanelDomain) {
					c.Header("Access-Control-Allow-Origin", origin)
					allowed = true
				}

				if !allowed {
					originHost := origin
					if strings.HasPrefix(originHost, "http://") {
						originHost = strings.TrimPrefix(originHost, "http://")
					} else if strings.HasPrefix(originHost, "https://") {
						originHost = strings.TrimPrefix(originHost, "https://")
					}
					if idx := strings.Index(originHost, ":"); idx != -1 {
						originHost = originHost[:idx]
					}

					hostOnly := host
					if idx := strings.Index(hostOnly, ":"); idx != -1 {
						hostOnly = hostOnly[:idx]
					}

					if originHost == hostOnly || originHost == "localhost" || originHost == "127.0.0.1" {
						c.Header("Access-Control-Allow-Origin", origin)
						allowed = true
					}
				}
			}
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