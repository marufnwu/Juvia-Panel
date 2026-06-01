package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHashToken_SHA256(t *testing.T) {
	token := "test-refresh-token-12345"

	hash := hashToken(token)

	assert.NotEmpty(t, hash)
	assert.NotEqual(t, token, hash, "hash should not equal input")

	// Verify it matches manual SHA-256
	expected := sha256.Sum256([]byte(token))
	expectedHex := hex.EncodeToString(expected[:])
	assert.Equal(t, expectedHex, hash)
}

func TestHashToken_Deterministic(t *testing.T) {
	token := "consistent-token"

	hash1 := hashToken(token)
	hash2 := hashToken(token)

	assert.Equal(t, hash1, hash2, "same input should produce same hash")
}

func TestHashToken_DifferentInputs(t *testing.T) {
	hash1 := hashToken("token-a")
	hash2 := hashToken("token-b")

	assert.NotEqual(t, hash1, hash2, "different inputs should produce different hashes")
}

func TestHashToken_EmptyString(t *testing.T) {
	hash := hashToken("")
	assert.NotEmpty(t, hash)

	expected := sha256.Sum256([]byte(""))
	expectedHex := hex.EncodeToString(expected[:])
	assert.Equal(t, expectedHex, hash)
}

func TestGenerateSessionID_Uniqueness(t *testing.T) {
	id1, err := generateSessionID()
	assert.NoError(t, err)

	id2, err := generateSessionID()
	assert.NoError(t, err)

	assert.NotEqual(t, id1, id2, "session IDs should be unique")
	assert.Len(t, id1, 64, "session ID should be 64 hex chars (32 bytes)")
}

func TestGenerateBackupCodes(t *testing.T) {
	codes, err := generateBackupCodes(8)
	assert.NoError(t, err)
	assert.Len(t, codes, 8)

	for _, code := range codes {
		assert.Len(t, code, 8, "each backup code should be 8 chars")
		assert.Equal(t, code, code, "backup codes should be uppercase")
	}

	// Check uniqueness
	seen := make(map[string]bool)
	for _, code := range codes {
		assert.False(t, seen[code], "backup code %s should be unique", code)
		seen[code] = true
	}
}
