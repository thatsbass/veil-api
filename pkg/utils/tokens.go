package utils

import "strings"

// EstimateTokens gives a rough token count (≈4 chars per token).
// Used only for quota pre-checks; actual counts come from provider responses.
func EstimateTokens(text string) int {
	if text == "" {
		return 0
	}
	words := len(strings.Fields(text))
	// average 1.3 tokens per word is a reasonable heuristic
	return int(float64(words) * 1.3)
}
