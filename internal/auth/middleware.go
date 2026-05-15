package auth

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"
)

const (
	userContextKey  = "user"
	dashboardSessionKey = "dashboard_session"
)

func extractBearerToken(c *fiber.Ctx) string {
	header := c.Get(fiber.HeaderAuthorization)
	if after, ok := strings.CutPrefix(header, "Bearer "); ok {
		return after
	}
	return ""
}

// DashboardAuthMiddleware verifies session JWTs via the injected AuthProvider.
// Protects /api/* — used by the Next.js dashboard only.
func DashboardAuthMiddleware(provider AuthProvider) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := extractBearerToken(c)
		if token == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "missing token",
			})
		}
		user, err := provider.VerifyToken(c.Context(), token)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid session",
			})
		}
		c.Locals(dashboardSessionKey, user)
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
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"error": err.Error(),
		})
	case errors.Is(err, ErrInvalidAPIKey),
		errors.Is(err, ErrMissingAPIKey),
		errors.Is(err, ErrInvalidKeyFormat),
		errors.Is(err, ErrInvalidKeyLength):
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "invalid api key",
		})
	default:
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "internal error",
		})
	}
}
