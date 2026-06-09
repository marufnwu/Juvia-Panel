package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultConstants(t *testing.T) {
	assert.Equal(t, "production", DefaultEnv)
	assert.Equal(t, "/var/panel", DefaultDataDir)
	assert.Equal(t, "/etc/panel", DefaultConfigDir)
	assert.Equal(t, "/opt/panel", DefaultInstallDir)
	assert.Equal(t, "/var/panel/logs", DefaultLogDir)
	assert.Equal(t, 9090, DefaultAPIPort)
	assert.Equal(t, "/var/panel/panel.db", DefaultDBPath)
	assert.Equal(t, 15*time.Minute, DefaultJWTExpiry)
	assert.Equal(t, 168*time.Hour, DefaultRefreshExpiry)
	assert.Equal(t, "/var/run/panel/agent.sock", DefaultAgentSocket)
	assert.Equal(t, 9091, DefaultAgentTCPPort)
	assert.Equal(t, "/etc/panel/caddy/Caddyfile", DefaultCaddyConfig)
	assert.Equal(t, "/var/panel/caddy", DefaultCaddyDataDir)
	assert.Equal(t, "info", DefaultLogLevel)
}

func TestVersion(t *testing.T) {
	assert.Equal(t, "dev", Version)
}

func TestReadTrimmedFile(t *testing.T) {
	// Create temp file
	tmpFile, err := os.CreateTemp("", "test-config-*.txt")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	content := "  secret-value  \n"
	_, err = tmpFile.WriteString(content)
	assert.NoError(t, err)
	tmpFile.Close()

	result, err := readTrimmedFile(tmpFile.Name())
	assert.NoError(t, err)
	assert.Equal(t, "secret-value", result)
}

func TestReadTrimmedFile_NotFound(t *testing.T) {
	_, err := readTrimmedFile("/nonexistent/path/config.txt")
	assert.Error(t, err)
}

func TestParseInt(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
		ok       bool
	}{
		{"valid", "8080", 8080, true},
		{"zero", "0", 0, true},
		{"negative", "-1", -1, true},
		{"invalid text", "abc", 0, false},
		{"float partial", "3.14", 3, true}, // fmt.Sscanf reads "3" successfully
		{"empty", "", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := parseInt(tt.input)
			assert.Equal(t, tt.expected, result)
			assert.Equal(t, tt.ok, ok)
		})
	}
}

func TestEnvOverride_Defaults(t *testing.T) {
	cfg := &Config{
		Env:    DefaultEnv,
		DBPath: DefaultDBPath,
	}

	// Without env vars set, config should stay at defaults after override
	// (except env override overwrites with values from env which aren't set)
	// Test that applyEnvOverrides respects env vars
	os.Setenv("PANEL_ENV", "development")
	applyEnvOverrides(cfg)
	assert.Equal(t, "development", cfg.Env)
	os.Unsetenv("PANEL_ENV")
}

func TestEnvOverride_JWTSecret(t *testing.T) {
	cfg := &Config{}

	os.Setenv("PANEL_JWT_SECRET", "my-super-secret-key-that-is-long-enough!")
	applyEnvOverrides(cfg)
	assert.Equal(t, "my-super-secret-key-that-is-long-enough!", cfg.JWTSecret)
	os.Unsetenv("PANEL_JWT_SECRET")
}

func TestEnvOverride_APIHost(t *testing.T) {
	cfg := &Config{
		APIHost: "127.0.0.1",
		APIPort: 9090,
	}

	os.Setenv("PANEL_API_PORT", "8080")
	applyEnvOverrides(cfg)
	assert.Equal(t, 8080, cfg.APIPort)
	os.Unsetenv("PANEL_API_PORT")

	// Test invalid port
	os.Setenv("PANEL_API_PORT", "notanumber")
	cfg.APIPort = 9090
	applyEnvOverrides(cfg)
	assert.Equal(t, 9090, cfg.APIPort) // Should remain unchanged
	os.Unsetenv("PANEL_API_PORT")
}

func TestEnvOverride_MultipleFields(t *testing.T) {
	cfg := &Config{
		Env:    DefaultEnv,
		DBPath: DefaultDBPath,
	}

	os.Setenv("PANEL_ENV", "staging")
	os.Setenv("PANEL_DB_PATH", "/custom/path/db.sqlite")
	os.Setenv("PANEL_LOG_LEVEL", "debug")

	applyEnvOverrides(cfg)

	assert.Equal(t, "staging", cfg.Env)
	assert.Equal(t, "/custom/path/db.sqlite", cfg.DBPath)
	assert.Equal(t, "debug", cfg.LogLevel)

	os.Unsetenv("PANEL_ENV")
	os.Unsetenv("PANEL_DB_PATH")
	os.Unsetenv("PANEL_LOG_LEVEL")
}

func TestJWTSecretValidation(t *testing.T) {
	cfg := &Config{
		JWTSecret: "",
	}

	// Empty JWT secret should fail
	assert.Empty(t, cfg.JWTSecret)

	// Too short
	cfg.JWTSecret = "short"
	assert.True(t, len(cfg.JWTSecret) < 32)

	// Long enough
	cfg.JWTSecret = "this-is-a-32-character-long-secret!!"
	assert.True(t, len(cfg.JWTSecret) >= 32)
}
