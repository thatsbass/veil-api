package auth

import (
	"github.com/gofiber/fiber/v2"
	"github.com/thatsbass/veil/pkg/models"
)

// APIKeyMiddleware validates vl_live_xxx keys.
// Protects /v1/* — used by LLM clients (Claude CLI, Cursor, Aider, etc.).
func APIKeyMiddleware(svc AuthService) fiber.Handler {
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

// APIKeyUserFrom extracts the authenticated user from a /v1/* request context.
func APIKeyUserFrom(c *fiber.Ctx) *models.User {
	u, _ := c.Locals(userContextKey).(*models.User)
	return u
}
