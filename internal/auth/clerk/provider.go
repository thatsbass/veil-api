// Package clerk implements auth.AuthProvider using Clerk as the identity backend.
// Nothing outside this package imports clerk-sdk-go directly.
package clerk

import (
	"context"
	"fmt"
	"time"

	clerklib "github.com/clerk/clerk-sdk-go/v2"
	clerkjwt "github.com/clerk/clerk-sdk-go/v2/jwt"
	clerkuser "github.com/clerk/clerk-sdk-go/v2/user"

	"github.com/thatsbass/veil/internal/auth"
)

type provider struct{}

// NewProvider returns an auth.AuthProvider backed by Clerk.
// Call this once at startup and inject it via the auth.AuthProvider interface.
func NewProvider(secretKey string) auth.AuthProvider {
	clerklib.SetKey(secretKey)
	return &provider{}
}

func (p *provider) VerifyToken(ctx context.Context, token string) (*auth.AuthUser, error) {
	claims, err := clerkjwt.Verify(ctx, &clerkjwt.VerifyParams{Token: token})
	if err != nil {
		return nil, fmt.Errorf("clerk.VerifyToken: %w", err)
	}
	return p.GetUser(ctx, claims.Subject)
}

func (p *provider) GetUser(ctx context.Context, userID string) (*auth.AuthUser, error) {
	u, err := clerkuser.Get(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("clerk.GetUser: %w", err)
	}

	email := ""
	for _, e := range u.EmailAddresses {
		if u.PrimaryEmailAddressID != nil && e.ID == *u.PrimaryEmailAddressID {
			email = e.EmailAddress
			break
		}
	}

	return &auth.AuthUser{
		ID:        u.ID,
		Email:     email,
		CreatedAt: time.Unix(u.CreatedAt, 0).UTC(),
	}, nil
}
