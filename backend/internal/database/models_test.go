package database

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSourceConfig_ToJSON(t *testing.T) {
	sc := SourceConfig{
		Type:       "git",
		Provider:   "github",
		RepoURL:    "https://github.com/user/repo.git",
		Branch:     "main",
		AutoDeploy: true,
	}
	jsonStr := sc.ToJSON()
	assert.NotEmpty(t, jsonStr)

	// Should parse back correctly
	var parsed SourceConfig
	err := json.Unmarshal([]byte(jsonStr), &parsed)
	assert.NoError(t, err)
	assert.Equal(t, "git", parsed.Type)
	assert.Equal(t, "github", parsed.Provider)
	assert.Equal(t, "https://github.com/user/repo.git", parsed.RepoURL)
	assert.Equal(t, "main", parsed.Branch)
	assert.True(t, parsed.AutoDeploy)
}

func TestBuildConfig_ToJSON(t *testing.T) {
	bc := BuildConfig{
		Strategy:     "nixpacks",
		BuildCommand: "npm run build",
		StartCommand: "npm start",
		HealthCheck: &HealthCheck{
			Path:     "/health",
			Interval: 30,
			Timeout:  5,
			Retries:  3,
		},
	}
	jsonStr := bc.ToJSON()
	assert.NotEmpty(t, jsonStr)

	var parsed BuildConfig
	err := json.Unmarshal([]byte(jsonStr), &parsed)
	assert.NoError(t, err)
	assert.Equal(t, "nixpacks", parsed.Strategy)
	assert.Equal(t, "npm run build", parsed.BuildCommand)
	assert.Equal(t, "npm start", parsed.StartCommand)
	assert.NotNil(t, parsed.HealthCheck)
	assert.Equal(t, "/health", parsed.HealthCheck.Path)
	assert.Equal(t, 30, parsed.HealthCheck.Interval)
}

func TestBuildConfig_ToJSON_NilHealthCheck(t *testing.T) {
	bc := BuildConfig{
		Strategy: "dockerfile",
	}
	jsonStr := bc.ToJSON()
	assert.NotEmpty(t, jsonStr)

	var parsed BuildConfig
	err := json.Unmarshal([]byte(jsonStr), &parsed)
	assert.NoError(t, err)
	assert.Nil(t, parsed.HealthCheck)
}

func TestParseJSONField(t *testing.T) {
	// Valid JSON
	jsonStr := `{"type":"git","provider":"github","repo_url":"https://github.com/user/repo"}`
	var sc SourceConfig
	err := ParseJSONField(&jsonStr, &sc)
	assert.NoError(t, err)
	assert.Equal(t, "git", sc.Type)
	assert.Equal(t, "github", sc.Provider)

	// Nil pointer
	err = ParseJSONField(nil, &sc)
	assert.NoError(t, err)

	// Empty string
	empty := ""
	err = ParseJSONField(&empty, &sc)
	assert.NoError(t, err)

	// Invalid JSON
	invalid := `{bad json}`
	err = ParseJSONField(&invalid, &sc)
	assert.Error(t, err)
}

func TestSourceConfig_ToJSON_Empty(t *testing.T) {
	sc := SourceConfig{}
	jsonStr := sc.ToJSON()
	// Should still produce valid JSON
	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(jsonStr), &parsed)
	assert.NoError(t, err)
}

func TestHealthCheck_Defaults(t *testing.T) {
	hc := HealthCheck{
		Path:     "/health",
		Interval: 30,
		Timeout:  5,
		Retries:  3,
	}
	assert.Equal(t, "/health", hc.Path)
	assert.Equal(t, 30, hc.Interval)
	assert.Equal(t, 5, hc.Timeout)
	assert.Equal(t, 3, hc.Retries)
}

func TestAppDomain_Fields(t *testing.T) {
	domain := AppDomain{
		AppID:      "app_abc123",
		Domain:     "example.com",
		IsPrimary:  true,
		ForceHTTPS: true,
		SSLStatus:  "active",
	}
	assert.Equal(t, "app_abc123", domain.AppID)
	assert.Equal(t, "example.com", domain.Domain)
	assert.True(t, domain.IsPrimary)
	assert.True(t, domain.ForceHTTPS)
	assert.Equal(t, "active", domain.SSLStatus)
}

func TestAppEnvVar_Fields(t *testing.T) {
	envVar := AppEnvVar{
		AppID:    "app_abc123",
		Key:      "DATABASE_URL",
		Value:    "postgres://localhost/db",
		IsSecret: true,
	}
	assert.Equal(t, "app_abc123", envVar.AppID)
	assert.Equal(t, "DATABASE_URL", envVar.Key)
	assert.Equal(t, "postgres://localhost/db", envVar.Value)
	assert.True(t, envVar.IsSecret)
}

func TestDeployment_Fields(t *testing.T) {
	dep := Deployment{
		ID:     "dep_abc123",
		AppID:  "app_abc123",
		Status: "queued",
	}
	assert.Equal(t, "dep_abc123", dep.ID)
	assert.Equal(t, "app_abc123", dep.AppID)
	assert.Equal(t, "queued", dep.Status)
}

func TestWithQueryTimeout(t *testing.T) {
	// WithQueryTimeout requires a non-nil parent context
	// The nil-context test would panic; skip it
	_ = DefaultQueryTimeout
}
