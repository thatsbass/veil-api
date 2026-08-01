package auth

import (
	"crypto/rand"
	"math/big"
	"strings"
)

// alphabet excludes visually ambiguous characters: 0, 1, O, I, L.
const alphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"

// generateUserCode returns a human-readable code in the form XXXX-XXXX.
func generateUserCode() (string, error) {
	n := big.NewInt(int64(len(alphabet)))
	var parts [2]string
	for i := range parts {
		var b strings.Builder
		for j := 0; j < 4; j++ {
			idx, err := rand.Int(rand.Reader, n)
			if err != nil {
				return "", err
			}
			b.WriteByte(alphabet[idx.Int64()])
		}
		parts[i] = b.String()
	}
	return parts[0] + "-" + parts[1], nil
}
