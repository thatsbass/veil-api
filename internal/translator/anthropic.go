package translator

import (
	"fmt"
	"io"

	"github.com/gofiber/fiber/v2"
	"github.com/thatsbass/veil/pkg/models"
)

type anthropicTranslator struct{}

// NewAnthropic handles requests from Claude CLI / Claude SDK.
func NewAnthropic() Translator {
	return &anthropicTranslator{}
}

// anthropicRequest mirrors the Anthropic Messages API body.
type anthropicRequest struct {
	Model     string           `json:"model"`
	Messages  []models.Message `json:"messages"`
	System    string           `json:"system,omitempty"`
	MaxTokens int              `json:"max_tokens"`
	Stream    bool             `json:"stream,omitempty"`
	Tools     []models.Tool    `json:"tools,omitempty"`

	Temperature *float64 `json:"temperature,omitempty"`
	TopP        *float64 `json:"top_p,omitempty"`
}

func (t *anthropicTranslator) ParseRequest(c *fiber.Ctx) (*models.CompletionRequest, error) {
	var body anthropicRequest
	if err := c.BodyParser(&body); err != nil {
		return nil, fmt.Errorf("translator.anthropic.ParseRequest: %w", err)
	}
	return &models.CompletionRequest{
		Config: models.ModelConfig{
			Model:       body.Model,
			MaxTokens:   body.MaxTokens,
			Temperature: body.Temperature,
			TopP:        body.TopP,
			Stream:      body.Stream,
		},
		Payload: models.RequestPayload{
			Messages: body.Messages,
			System:   body.System,
			Tools:    body.Tools,
		},
	}, nil
}

func (t *anthropicTranslator) WriteResponse(c *fiber.Ctx, resp *models.CompletionResponse) error {
	return c.JSON(fiber.Map{
		"id":          resp.ID,
		"type":        "message",
		"role":        "assistant",
		"model":       resp.Model,
		"content":     resp.Content,
		"stop_reason": resp.StopReason,
		"usage": fiber.Map{
			"input_tokens":  resp.Usage.InputTokens,
			"output_tokens": resp.Usage.OutputTokens,
		},
	})
}

func (t *anthropicTranslator) WriteStreamEvent(w io.Writer, event models.StreamEvent) error {
	return marshalSSE(w, event)
}
