package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
)

// tokenBytes is the amount of entropy in an opaque session token.
const tokenBytes = 32

// GenerateToken returns a new random opaque token (URL-safe, unpadded).
// The plaintext is shown to the client once; only its hash is stored.
func GenerateToken() (string, error) {
	b := make([]byte, tokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// HashToken returns the hex-encoded SHA-256 of a token, for storage and lookup.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
