package auth

import "github.com/gofiber/fiber/v2"

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
