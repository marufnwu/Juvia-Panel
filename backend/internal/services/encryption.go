package services

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

// Encryption service for AES-256-GCM encryption
// Uses PANEL_MASTER_KEY environment variable as the key

// Encrypt encrypts plaintext using AES-256-GCM
func Encrypt(plaintext string, masterKey string) (string, error) {
	if masterKey == "" {
		return "", errors.New("master key is required for encryption")
	}

	// Derive a 32-byte key from the master key using a simple hash
	// In production, use a proper KDF like Argon2 or PBKDF2
	key := deriveKey(masterKey)

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts ciphertext encrypted with Encrypt
func Decrypt(ciphertext string, masterKey string) (string, error) {
	if masterKey == "" {
		return "", errors.New("master key is required for decryption")
	}

	// Derive the same 32-byte key
	key := deriveKey(masterKey)

	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

// IsSecretKey returns true if the key name suggests it contains sensitive data
func IsSecretKey(key string) bool {
	upperKey := key
	secretIndicators := []string{"SECRET", "KEY", "PASSWORD", "TOKEN", "PRIVATE", "CREDENTIAL", "AUTH"}
	for _, indicator := range secretIndicators {
		if contains(upperKey, indicator) {
			return true
		}
	}
	return false
}

// MaskSecret masks a secret value for display
func MaskSecret(value string) string {
	if len(value) <= 8 {
		return "********"
	}
	return value[:4] + "..." + value[len(value)-4:]
}

// contains checks if substr exists in s (case-sensitive)
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// deriveKey derives a 32-byte key from the master key using SHA-256
func deriveKey(masterKey string) []byte {
	h := sha256.Sum256([]byte(masterKey))
	return h[:]
}
