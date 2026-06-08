package handlers

import (
	"net/http"
	"time"

	"panel-api/internal/config"

	"github.com/gin-gonic/gin"
)

// HealthCheck handles GET /health endpoint.
func HealthCheck(c *gin.Context) {
	cfg, _ := c.Get("config")
	version := "unknown"
	if cfg != nil {
		version = cfg.(*config.Config).Version
	}
	c.JSON(http.StatusOK, gin.H{
		"status":     "ok",
		"version":   version,
		"time":      time.Now().UTC().Format(time.RFC3339),
		"request_id": c.GetString("request_id"),
	})
}
