package translator

import (
	"fmt"
	"io"

	"github.com/gofiber/fiber/v2"
	"github.com/thatsbass/veil/pkg/models"
)

type openAITranslator struct{}

// NewOpenAI handles requests from OpenAI-compatible clients (Cursor, Aider, etc.).
func NewOpenAI() Translator {
	return &openAITranslator{}
}

// openAIRequest mirrors the OpenAI Chat Completions body.
type openAIRequest struct {
	Model       string           `json:"model"`
	Messages    []models.Message `json:"messages"`
	MaxTokens   int              `json:"max_tokens,omitempty"`
	Temperature *float64         `json:"temperature,omitempty"`
	TopP        *float64         `json:"top_p,omitempty"`
	Stream      bool             `json:"stream,omitempty"`
}

func (t *openAITranslator) ParseRequest(c *fiber.Ctx) (*models.CompletionRequest, error) {
	var body openAIRequest
	if err := c.BodyParser(&body); err != nil {
		return nil, fmt.Errorf("translator.openai.ParseRequest: %w", err)
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
		},
	}, nil
}

func (t *openAITranslator) WriteResponse(c *fiber.Ctx, resp *models.CompletionResponse) error {
	content := ""
	if len(resp.Content) > 0 {
		content = resp.Content[0].Text
	}
	return c.JSON(fiber.Map{
		"id":    resp.ID,
		"model": resp.Model,
		"choices": []fiber.Map{{
			"index": 0,
			"message": fiber.Map{
				"role":    "assistant",
				"content": content,
			},
			"finish_reason": resp.StopReason,
		}},
		"usage": fiber.Map{
			"prompt_tokens":     resp.Usage.InputTokens,
			"completion_tokens": resp.Usage.OutputTokens,
			"total_tokens":      resp.Usage.InputTokens + resp.Usage.OutputTokens,
		},
	})
}

func (t *openAITranslator) WriteStreamEvent(w io.Writer, event models.StreamEvent) error {
	return marshalSSE(w, event)
}
