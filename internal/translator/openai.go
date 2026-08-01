package translator

import (
	"bufio"
	"encoding/json"
	"fmt"

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

// StreamEvents converts Anthropic SSE events to OpenAI Chat Completions streaming
// format expected by Cursor / Aider, then writes [DONE] to close the stream.
func (t *openAITranslator) StreamEvents(w *bufio.Writer, events <-chan models.StreamEvent) error {
	for event := range events {
		var err error
		switch event.Type {
		case "content_block_delta":
			text := extractTextDelta(event.Delta)
			if text == "" {
				continue
			}
			err = marshalSSE(w, fiber.Map{
				"choices": []fiber.Map{{
					"index":         0,
					"delta":         fiber.Map{"content": text},
					"finish_reason": nil,
				}},
			})
		case "message_delta":
			stopReason := extractStopReason(event.Delta)
			if stopReason == "" {
				continue
			}
			err = marshalSSE(w, fiber.Map{
				"choices": []fiber.Map{{
					"index":         0,
					"delta":         fiber.Map{},
					"finish_reason": stopReason,
				}},
			})
		default:
			continue
		}
		if err != nil {
			return err
		}
		_ = w.Flush()
	}
	_, _ = fmt.Fprintf(w, "data: [DONE]\n\n")
	_ = w.Flush()
	return nil
}

// extractTextDelta pulls the text string from an Anthropic text_delta JSON blob.
func extractTextDelta(raw json.RawMessage) string {
	if raw == nil {
		return ""
	}
	var d struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &d); err != nil || d.Type != "text_delta" {
		return ""
	}
	return d.Text
}

// extractStopReason pulls stop_reason from an Anthropic message_delta JSON blob.
func extractStopReason(raw json.RawMessage) string {
	if raw == nil {
		return ""
	}
	var d struct {
		StopReason string `json:"stop_reason"`
	}
	if err := json.Unmarshal(raw, &d); err != nil {
		return ""
	}
	return d.StopReason
}
