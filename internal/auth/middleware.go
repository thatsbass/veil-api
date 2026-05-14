package auth

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"
)

const userContextKey = "user"

// Middleware returns a Fiber handler that validates the Bearer token and
// stores the resolved user in c.Locals(userContextKey).
func Middleware(svc AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		rawKey := extractBearerToken(c)
		user, err := svc.ValidateKey(c.Context(), rawKey)
		if err != nil {
			return respondAuthError(c, err)
		}
		c.Locals(userContextKey, user)
		return c.Next()
	}
}

func extractBearerToken(c *fiber.Ctx) string {
	header := c.Get(fiber.HeaderAuthorization)
	if after, ok := strings.CutPrefix(header, "Bearer "); ok {
		return after
	}
	return ""
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
