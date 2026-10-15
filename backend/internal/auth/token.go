package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// GenerateSessionToken creates a 32-byte cryptographically secure random token
// and returns both the hex string for client cookie and the SHA-256 hash for database storage.
func GenerateSessionToken() (string, []byte, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", nil, fmt.Errorf("generating random session bytes: %w", err)
	}

	token := hex.EncodeToString(bytes)
	hash := HashToken(token)
	return token, hash, nil
}

// HashToken calculates the SHA-256 digest of a session token string.
func HashToken(token string) []byte {
	hash := sha256.Sum256([]byte(token))
	return hash[:]
}
