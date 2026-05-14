package translator

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/gofiber/fiber/v2"
	"github.com/thatsbass/veil/pkg/models"
)

type responsesTranslator struct{}

// NewResponses handles requests from the OpenAI Responses API (Codex CLI).
func NewResponses() Translator {
	return &responsesTranslator{}
}

// responsesRequest mirrors the OpenAI Responses API body.
type responsesRequest struct {
	Model       string           `json:"model"`
	Input       []models.Message `json:"input"`
	MaxTokens   int              `json:"max_output_tokens,omitempty"`
	Temperature *float64         `json:"temperature,omitempty"`
	Stream      bool             `json:"stream,omitempty"`
}

func (t *responsesTranslator) ParseRequest(c *fiber.Ctx) (*models.CompletionRequest, error) {
	var body responsesRequest
	if err := c.BodyParser(&body); err != nil {
		return nil, fmt.Errorf("translator.responses.ParseRequest: %w", err)
	}
	return &models.CompletionRequest{
		Config: models.ModelConfig{
			Model:       body.Model,
			MaxTokens:   body.MaxTokens,
			Temperature: body.Temperature,
			Stream:      body.Stream,
		},
		Payload: models.RequestPayload{
			Messages: body.Input,
		},
	}, nil
}

func (t *responsesTranslator) WriteResponse(c *fiber.Ctx, resp *models.CompletionResponse) error {
	content := ""
	if len(resp.Content) > 0 {
		content = resp.Content[0].Text
	}
	return c.JSON(fiber.Map{
		"id":    resp.ID,
		"model": resp.Model,
		"output": []fiber.Map{{
			"type":    "message",
			"role":    "assistant",
			"content": content,
		}},
		"usage": fiber.Map{
			"input_tokens":  resp.Usage.InputTokens,
			"output_tokens": resp.Usage.OutputTokens,
		},
	})
}

func (t *responsesTranslator) WriteStreamEvent(w io.Writer, event models.StreamEvent) error {
	return marshalSSE(w, event)
}

// marshalSSE writes a value as "data: <json>\n\n" to w.
func marshalSSE(w io.Writer, v any) error {
	raw, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshalSSE: %w", err)
	}
	_, err = fmt.Fprintf(w, "data: %s\n\n", raw)
	return err
}
