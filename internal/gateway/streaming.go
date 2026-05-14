package gateway

import (
	"bufio"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/thatsbass/veil/internal/translator"
	"github.com/thatsbass/veil/pkg/models"
)

// streamResponse sets SSE headers then pumps events from the channel to the
// client using Fiber's body stream writer.
func streamResponse(c *fiber.Ctx, t translator.Translator, events <-chan models.StreamEvent) error {
	c.Set(fiber.HeaderContentType, "text/event-stream")
	c.Set(fiber.HeaderCacheControl, "no-cache")
	c.Set("X-Accel-Buffering", "no")

	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		for event := range events {
			if err := t.WriteStreamEvent(w, event); err != nil {
				return
			}
			_ = w.Flush()
		}
		_, _ = fmt.Fprintf(w, "data: [DONE]\n\n")
		_ = w.Flush()
	})
	return nil
}
