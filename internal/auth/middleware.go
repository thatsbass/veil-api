package auth

import (
	"context"
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/thatsbass/veil/pkg/models"
)

const (
	userContextKey      = "user"
	dashboardSessionKey = "dashboard_session"
)

// UserProvisioner creates or updates a user record on first sign-in.
type UserProvisioner interface {
	ResolveUser(ctx context.Context, email string) (*models.User, error)
}

func extractBearerToken(c *fiber.Ctx) string {
	header := c.Get(fiber.HeaderAuthorization)
	if after, ok := strings.CutPrefix(header, "Bearer "); ok {
		return after
	}
	return ""
}

// DashboardAuthMiddleware verifies session JWTs via the injected AuthProvider.
// If a provisioner is provided, it upserts the user in the database on every
// authenticated request (no-op after the first call thanks to ON CONFLICT).
// Protects /api/* — used by the React dashboard only.
func DashboardAuthMiddleware(provider AuthProvider, provisioner ...UserProvisioner) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := extractBearerToken(c)
		if token == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing token"})
		}
		session, err := provider.VerifyToken(c.Context(), token)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid session"})
		}
		c.Locals(dashboardSessionKey, session)

		if len(provisioner) > 0 && provisioner[0] != nil {
			// Non-fatal: upsert the user on first sign-in. Don't block the request on error.
			if _, provErr := provisioner[0].ResolveUser(c.Context(), session.Email); provErr != nil {
				_ = provErr
			}
		}

		return c.Next()
	}
}

// DashboardUserFrom extracts the verified AuthUser from a Fiber context.
// Returns nil if the route is not protected by DashboardAuthMiddleware.
func DashboardUserFrom(c *fiber.Ctx) *AuthUser {
	u, _ := c.Locals(dashboardSessionKey).(*AuthUser)
	return u
}

func respondAuthError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, ErrQuotaExceeded):
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"error": err.Error()})
	case errors.Is(err, ErrInvalidAPIKey),
		errors.Is(err, ErrMissingAPIKey),
		errors.Is(err, ErrInvalidKeyFormat),
		errors.Is(err, ErrInvalidKeyLength):
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid api key"})
	default:
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}
}
