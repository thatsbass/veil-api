package auth

import (
	"context"
	"time"
)

// AuthProvider is the provider-agnostic identity interface.
// Implementations live in sub-packages (e.g. internal/auth/clerk).
type AuthProvider interface {
	// VerifyToken validates a raw bearer token and returns the associated user.
	VerifyToken(ctx context.Context, token string) (*AuthUser, error)
	// GetUser fetches user details by provider-issued ID.
	GetUser(ctx context.Context, userID string) (*AuthUser, error)
}

// AuthUser holds the normalized identity returned by any AuthProvider.
// It deliberately contains no provider-specific types.
type AuthUser struct {
	ID        string
	Email     string
	CreatedAt time.Time
}
