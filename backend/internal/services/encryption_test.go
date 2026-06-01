package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	masterKey := "this-is-a-32-byte-master-key!!"
	plaintext := "hello world secret data"

	encrypted, err := Encrypt(plaintext, masterKey)
	require.NoError(t, err)
	assert.NotEmpty(t, encrypted)
	assert.NotEqual(t, plaintext, encrypted)

	decrypted, err := Decrypt(encrypted, masterKey)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestEncrypt_EmptyMasterKey(t *testing.T) {
	_, err := Encrypt("test", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "master key is required")
}

func TestDecrypt_EmptyMasterKey(t *testing.T) {
	_, err := Decrypt("dGVzdA==", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "master key is required")
}

func TestDecrypt_InvalidCiphertext(t *testing.T) {
	_, err := Decrypt("not-valid-base64!!!", "key")
	assert.Error(t, err)
}

func TestDecrypt_TooShortCiphertext(t *testing.T) {
	_, err := Decrypt("dGVzdA==", "this-is-a-32-byte-master-key!!")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ciphertext too short")
}

func TestDecrypt_WrongKey(t *testing.T) {
	masterKey := "this-is-a-32-byte-master-key!!"
	plaintext := "secret data"

	encrypted, err := Encrypt(plaintext, masterKey)
	require.NoError(t, err)

	_, err = Decrypt(encrypted, "wrong-key-that-is-32-bytes-long!!")
	assert.Error(t, err)
}

func TestEncrypt_DifferentCiphertextsSamePlaintext(t *testing.T) {
	masterKey := "this-is-a-32-byte-master-key!!"
	plaintext := "same data"

	enc1, err := Encrypt(plaintext, masterKey)
	require.NoError(t, err)

	enc2, err := Encrypt(plaintext, masterKey)
	require.NoError(t, err)

	assert.NotEqual(t, enc1, enc2, "same plaintext should produce different ciphertexts due to random nonce")
}

func TestDeriveKey_Deterministic(t *testing.T) {
	key1 := deriveKey("test-key")
	key2 := deriveKey("test-key")
	assert.Equal(t, key1, key2)
}

func TestDeriveKey_DifferentKeys(t *testing.T) {
	key1 := deriveKey("key-one")
	key2 := deriveKey("key-two")
	assert.NotEqual(t, key1, key2)
}

func TestDeriveKey_Length(t *testing.T) {
	key := deriveKey("any-key")
	assert.Len(t, key, 32, "derived key must be 32 bytes for AES-256")
}

func TestIsSecretKey(t *testing.T) {
	tests := []struct {
		key      string
		expected bool
	}{
		{"DATABASE_PASSWORD", true},
		{"API_SECRET_KEY", true},
		{"JWT_TOKEN", true},
		{"PRIVATE_KEY", true},
		{"CREDENTIALS", true},
		{"AUTH_TOKEN", true},
		{"APP_NAME", false},
		{"PORT", false},
		{"DEBUG", false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsSecretKey(tt.key))
		})
	}
}

func TestMaskSecret(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"short", "********"},
		{"12345678", "********"},
		{"123456789", "1234...6789"},
		{"abcdefghijklmnop", "abcd...mnop"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, MaskSecret(tt.input))
		})
	}
}
