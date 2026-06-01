package config

import (
	"fmt"
	"os"
	"time"
)

// Config holds all configuration for the application.
type Config struct {
	// Environment
	Env string

	// Directories
	DataDir   string
	ConfigDir string

	// Server
	APIPort int

	// Database
	DBPath string

	// JWT
	JWTSecret       string
	JWTExpiry       time.Duration
	RefreshExpiry   time.Duration

	// Master key for encryption (AES-256)
	MasterKey string

	// Panel domain
	PanelDomain string

	// Agent socket path
	AgentSocket string

	// Log level
	LogLevel string

	// Allowed CORS origins (comma-separated)
	AllowedOrigins string
}

// Load reads configuration from environment variables with defaults.
func Load() (*Config, error) {
	cfg := &Config{
		Env:         getEnv("PANEL_ENV", "development"),
		DataDir:     getEnv("PANEL_DATA_DIR", "/var/panel"),
		ConfigDir:   getEnv("PANEL_CONFIG_DIR", "/etc/panel"),
		APIPort:     getEnvInt("PANEL_API_PORT", 9090),
		DBPath:      getEnv("PANEL_DB_PATH", "/var/panel/panel.db"),
		JWTSecret:   getEnv("PANEL_JWT_SECRET", ""),
		JWTExpiry:   getEnvDuration("PANEL_JWT_EXPIRY", 15*time.Minute),
		RefreshExpiry: getEnvDuration("PANEL_REFRESH_EXPIRY", 168*time.Hour), // 7 days
		MasterKey:   getEnv("PANEL_MASTER_KEY", ""),
		PanelDomain: getEnv("PANEL_DOMAIN", ""),
		AgentSocket: getEnv("PANEL_AGENT_SOCKET", "/var/run/panel/agent.sock"),
		LogLevel:    getEnv("PANEL_LOG_LEVEL", "info"),
		AllowedOrigins: getEnv("PANEL_ALLOWED_ORIGINS", ""),
	}

	// Validate required fields
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("PANEL_JWT_SECRET is required")
	}

	if len(cfg.JWTSecret) < 32 {
		return nil, fmt.Errorf("PANEL_JWT_SECRET must be at least 32 characters")
	}

	return cfg, nil
}

// getEnv returns the value of an environment variable or a default value.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt returns the integer value of an environment variable or a default value.
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var intValue int
		if _, err := fmt.Sscanf(value, "%d", &intValue); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvDuration returns the duration value of an environment variable or a default value.
func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
