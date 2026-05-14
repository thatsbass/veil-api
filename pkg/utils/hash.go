package utils

import (
	"crypto/sha256"
	"encoding/hex"
)

// HashAPIKey returns the hex-encoded SHA-256 hash of a raw API key.
// Store only this hash — never the plaintext key.
func HashAPIKey(rawKey string) string {
	sum := sha256.Sum256([]byte(rawKey))
	return hex.EncodeToString(sum[:])
}
