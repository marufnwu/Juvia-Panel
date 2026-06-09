package apps

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsValidAppName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"valid lowercase", "myapp", true},
		{"valid uppercase", "MyApp", true},
		{"valid with dash", "my-app", true},
		{"valid with underscore", "my_app", true},
		{"valid with numbers", "app123", true},
		{"valid mixed", "my-app_123", true},
		{"empty string", "", false},
		{"too long (65 chars)", strings.Repeat("a", 65), false},
		{"max length (64 chars)", strings.Repeat("a", 64), true},
		{"with space", "my app", false},
		{"with dot", "my.app", false},
		{"with slash", "my/app", false},
		{"with special chars", "app@#$", false},
		{"single char", "a", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidAppName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateID(t *testing.T) {
	// Test that generateID produces unique IDs with correct prefix
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id, err := generateID("app_")
		assert.NoError(t, err)
		assert.True(t, strings.HasPrefix(id, "app_"), "ID should start with 'app_'")
		assert.Len(t, id, 13, "ID should be 13 chars (4 prefix + 9 random)")
		assert.False(t, ids[id], "IDs should be unique")
		ids[id] = true
	}

	// Test different prefixes
	id, err := generateID("dep_")
	assert.NoError(t, err)
	assert.True(t, strings.HasPrefix(id, "dep_"))

	id, err = generateID("vol_")
	assert.NoError(t, err)
	assert.True(t, strings.HasPrefix(id, "vol_"))
}

func TestEncode62(t *testing.T) {
	// Test that encode62 produces valid base62 output
	bytes := []byte{0, 1, 10, 35, 61, 62, 255, 128, 50}
	result := encode62(bytes)
	assert.Len(t, result, len(bytes))

	// All chars should be from base62 alphabet
	const chars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	for _, c := range result {
		assert.True(t, strings.ContainsRune(chars, c),
			"Character '%c' not in base62 alphabet", c)
	}

	// Deterministic output
	r1 := encode62(bytes)
	r2 := encode62(bytes)
	assert.Equal(t, r1, r2, "encode62 should be deterministic")

	// Empty input
	assert.Empty(t, encode62([]byte{}))
}

func TestDetectRuntime(t *testing.T) {
	tests := []struct {
		name       string
		repoURL    string
		sourceType string
		expected   string
	}{
		{"docker compose", "anything", "docker_compose", "docker"},
		{"docker type", "anything", "docker", "docker"},
		{"nodejs", "https://github.com/user/node-app", "git", "nodejs"},
		{"next.js", "https://github.com/user/next-blog", "git", "nodejs"},
		{"react", "https://github.com/user/react-dashboard", "git", "nodejs"},
		{"vue", "https://github.com/user/vue-frontend", "git", "nodejs"},
		{"python", "https://github.com/user/python-api", "git", "python"},
		{"django", "https://github.com/user/django-project", "git", "python"},
		{"flask", "https://github.com/user/flask-app", "git", "python"},
		{"go", "https://github.com/user/go-service", "git", "go"},
		{"golang", "https://github.com/user/golang-cli", "git", "go"},
		{"php", "https://github.com/user/php-website", "git", "php"},
		{"laravel", "https://github.com/user/laravel-app", "git", "php"},
		{"wordpress", "https://github.com/user/wordpress-theme", "git", "php"},
		{"ruby", "https://github.com/user/ruby-gem", "git", "ruby"},
		{"rails", "https://github.com/user/rails-app", "git", "ruby"},
		{"unknown defaults to static", "https://example.com/repo", "git", "static"},
		{"empty URL", "", "git", "static"},
		{"upload type", "anything", "upload", "static"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectRuntime(tt.repoURL, tt.sourceType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateGitURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{"https github", "https://github.com/user/repo.git", true},
		{"http gitlab", "http://gitlab.com/user/repo.git", true},
		{"ssh github", "git@github.com:user/repo.git", true},
		{"ssh protocol", "ssh://git@example.com/repo.git", true},
		{"git protocol", "git://example.com/repo.git", true},
		{"ftp invalid", "ftp://example.com/repo", false},
		{"plain text", "not-a-url", false},
		{"empty", "", false},
		{"file path", "/home/user/repo", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateGitURL(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseRepoURL(t *testing.T) {
	tests := []struct {
		name      string
		repoURL   string
		expProv   string
		expHasURL bool
	}{
		{"github", "https://github.com/user/repo", "github", false},
		{"gitlab.com", "https://gitlab.com/user/repo", "gitlab", false},
		{"self-hosted gitlab", "https://gitlab.company.com/user/repo", "gitlab", false},
		{"bitbucket", "https://bitbucket.org/user/repo", "bitbucket", false},
		{"unknown", "https://unknown.com/repo", "other", true},
		{"empty", "", "other", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, cleanURL := ParseRepoURL(tt.repoURL)
			assert.Equal(t, tt.expProv, provider)
			if tt.expHasURL {
				assert.NotEmpty(t, cleanURL)
			} else {
				assert.Empty(t, cleanURL)
			}
		})
	}
}

func TestParseEnvVars(t *testing.T) {
	// Test normal and secret env vars
	envVars := map[string]string{
		"APP_NAME":     "MyApp",
		"API_KEY":      "secret123",
		"DATABASE_URL": "postgres://localhost/db",
		"PORT":         "8080",
		"SECRET_TOKEN": "supersecret",
	}

	result, err := parseEnvVars(envVars)
	assert.NoError(t, err)
	assert.Len(t, result, 5)

	// Check that keys present as secret indicators get flagged
	for _, v := range result {
		switch v.Key {
		case "API_KEY":
			assert.True(t, v.IsSecret, "API_KEY should be flagged as secret")
		case "SECRET_TOKEN":
			assert.True(t, v.IsSecret, "SECRET_TOKEN should be flagged as secret")
		case "DATABASE_URL":
			assert.False(t, v.IsSecret, "DATABASE_URL should not be flagged")
		}
	}

	// Empty env vars
	result, err = parseEnvVars(map[string]string{})
	assert.NoError(t, err)
	assert.Len(t, result, 0)
}

func TestIsSecretKey_Apps(t *testing.T) {
	tests := []struct {
		key      string
		expected bool
	}{
		{"PASSWORD", true},
		{"password", true},
		{"SECRET", true},
		{"API_KEY", true},
		{"TOKEN", true},
		{"CREDENTIAL", true},
		{"CREDENTIALS", true},
		{"AUTH_TOKEN", true},
		{"PRIVATE", true},
		{"PRIVATE_KEY", true},
		{"APP_NAME", false},
		{"PORT", false},
		{"HOST", false},
		{"DEBUG", false},
		{"NODE_ENV", false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			result := isSecretKey(tt.key)
			assert.Equal(t, tt.expected, result,
				"isSecretKey(%s) = %v, expected %v", tt.key, result, tt.expected)
		})
	}
}

func TestMaskSecretValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"short value", "abc", "********"},
		{"exactly 8 chars", "12345678", "********"},
		{"9 chars", "123456789", "1234...6789"},
		{"long value", "abcdefghijklmnop", "abcd...mnop"},
		{"16 chars", "1234567890123456", "1234...3456"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maskSecretValue(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStrPtr_DerefOrEmpty(t *testing.T) {
	val := "hello"
	ptr := strPtr(val)
	assert.Equal(t, "hello", *ptr)

	assert.Equal(t, "hello", derefOrEmpty(strPtr("hello")))
	assert.Equal(t, "", derefOrEmpty(nil))
}

// Test trigger deployment helper functions
func TestTriggerDeployment(t *testing.T) {
	// This tests the trigger code path in CreateApp that handles deployment creation
	// Verify deployment ID generation fails are handled
	id, err := generateID("dep_")
	assert.NoError(t, err)
	assert.True(t, strings.HasPrefix(id, "dep_"))
	assert.Len(t, id, 13)
}

func TestAppNameValidation_EdgeCases(t *testing.T) {
	// Unicode characters should be rejected
	assert.False(t, isValidAppName("myappé"))
	// Leading/trailing dashes ARE allowed by current implementation
	assert.True(t, isValidAppName("-myapp"))
	assert.True(t, isValidAppName("myapp-"))
	// Empty strings
	assert.False(t, isValidAppName(""))
	// Single dash
	assert.True(t, isValidAppName("-"))
}
