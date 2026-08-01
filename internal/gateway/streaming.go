package gateway

import (
	"bufio"

	"github.com/gofiber/fiber/v2"
	"github.com/thatsbass/veil/internal/translator"
	"github.com/thatsbass/veil/pkg/models"
)

// streamResponse sets SSE headers and delegates event writing to the translator.
func streamResponse(c *fiber.Ctx, t translator.Translator, events <-chan models.StreamEvent) error {
	c.Set(fiber.HeaderContentType, "text/event-stream")
	c.Set(fiber.HeaderCacheControl, "no-cache")
	c.Set("X-Accel-Buffering", "no")

	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		_ = t.StreamEvents(w, events)
	})
	return nil
}
