package domains

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsValidDomain(t *testing.T) {
	tests := []struct {
		name     string
		domain   string
		expected bool
	}{
		{"valid simple", "example.com", true},
		{"valid subdomain", "app.example.com", true},
		{"valid deep subdomain", "api.staging.example.com", true},
		{"valid with hyphen", "my-app.example.com", true},
		{"valid with numbers", "app123.example.com", true},
		{"too short (2 chars)", "ab", false},
		{"too long (254 chars)", makeString(254, 'a') + ".com", false},
		{"max length (253 chars)", makeString(249, 'a') + ".com", false},
		{"no dot", "localhost", false},
		{"single char parts", "a.b", true},
		{"part too long (64 chars)", makeString(64, 'a') + ".com", false},
		{"part max (63 chars)", makeString(63, 'a') + ".com", true},
		{"empty string", "", false},
		{"domain with trailing dot", "example.com.", false}, // trailing dot rejected by isValidDomain
		{"domain with leading dot", ".example.com", false},  // leading dot rejected by isValidDomain
		{"valid uppercase", "Example.COM", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidDomain(tt.domain)
			assert.Equal(t, tt.expected, result,
				"isValidDomain(%q) = %v, expected %v", tt.domain, result, tt.expected)
		})
	}
}

func TestIsValidDomainPart(t *testing.T) {
	tests := []struct {
		name     string
		part     string
		expected bool
	}{
		{"lowercase letters", "example", true},
		{"uppercase letters", "EXAMPLE", true},
		{"mixed case", "Example", true},
		{"with numbers", "app123", true},
		{"with hyphen", "my-app", true},
		{"with underscore", "my_app", true},
		{"empty", "", true}, // empty string passes the loop trivially
		{"with special chars", "app@!#", false},
		{"with space", "my app", false},
		{"single char", "a", true},
		{"hyphen only", "-", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidDomainPart(tt.part)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsValidDomain_RealWorldDomains(t *testing.T) {
	validDomains := []string{
		"github.com",
		"api.github.com",
		"www.example.co.uk",
		"my-app.staging.company.io",
		"sub.domain.example.org",
	}

	for _, d := range validDomains {
		t.Run(d, func(t *testing.T) {
			assert.True(t, isValidDomain(d), "Expected valid: %s", d)
		})
	}

	invalidDomains := []string{
		"",
		".",
		".com",
		"com",
		"https://example.com",
	}

	for _, d := range invalidDomains {
		t.Run("invalid_"+d, func(t *testing.T) {
			assert.False(t, isValidDomain(d), "Expected invalid: %s", d)
		})
	}
}

func makeString(n int, c byte) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = c
	}
	return string(b)
}
