package translator

import (
	"io"

	"github.com/gofiber/fiber/v2"
	"github.com/thatsbass/veil/pkg/models"
)

// Translator parses an inbound request and writes the outbound response,
// bridging between the caller's protocol and Veil's internal models.
type Translator interface {
	ParseRequest(c *fiber.Ctx) (*models.CompletionRequest, error)
	WriteResponse(c *fiber.Ctx, resp *models.CompletionResponse) error
	// WriteStreamEvent serialises one SSE chunk to w (a *bufio.Writer).
	WriteStreamEvent(w io.Writer, event models.StreamEvent) error
}
