package translator

import (
	"bufio"

	"github.com/gofiber/fiber/v2"
	"github.com/thatsbass/veil/pkg/models"
)

// Translator parses an inbound request and writes the outbound response,
// bridging between the caller's protocol and Veil's internal models.
type Translator interface {
	ParseRequest(c *fiber.Ctx) (*models.CompletionRequest, error)
	WriteResponse(c *fiber.Ctx, resp *models.CompletionResponse) error
	// StreamEvents reads all events from the channel and writes SSE to w,
	// including the terminal [DONE] marker. Implementations own flushing.
	StreamEvents(w *bufio.Writer, events <-chan models.StreamEvent) error
}
