package utils

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// GenerateAPIKey returns a new vl_live_ key with 30 bytes of randomness.
// Total length: 48 chars (8-char prefix + 40-char base64url body).
func GenerateAPIKey() (string, error) {
	b := make([]byte, 30)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("utils.GenerateAPIKey: %w", err)
	}
	return "vl_live_" + base64.RawURLEncoding.EncodeToString(b), nil
}
