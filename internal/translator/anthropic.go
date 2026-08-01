package translator

import (
	"bufio"
	"encoding/json"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/thatsbass/veil/pkg/models"
)

type anthropicTranslator struct{}

// NewAnthropic handles requests from Claude CLI / Claude SDK.
func NewAnthropic() Translator {
	return &anthropicTranslator{}
}

// anthropicRequest mirrors the Anthropic Messages API body.
// System is json.RawMessage because the API accepts both a plain string
// and an array of content blocks (Claude Code always sends the array form).
type anthropicRequest struct {
	Model     string           `json:"model"`
	Messages  []models.Message `json:"messages"`
	System    json.RawMessage  `json:"system,omitempty"`
	MaxTokens int              `json:"max_tokens"`
	Stream    bool             `json:"stream,omitempty"`
	Tools     []models.Tool    `json:"tools,omitempty"`
	Thinking  *thinkingConfig  `json:"thinking,omitempty"`

	Temperature *float64 `json:"temperature,omitempty"`
	TopP        *float64 `json:"top_p,omitempty"`
}

// thinkingConfig captures Claude's extended thinking configuration.
type thinkingConfig struct {
	Type         string `json:"type"`          // "enabled" | "disabled"
	BudgetTokens int    `json:"budget_tokens"`
}

// parseSystemField handles system as either a plain string or a content-block array.
func parseSystemField(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	// Try plain string first.
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	// Try array of content blocks: [{"type":"text","text":"..."}]
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &blocks); err == nil {
		out := ""
		for _, b := range blocks {
			if b.Type == "text" {
				out += b.Text
			}
		}
		return out
	}
	return ""
}

func (t *anthropicTranslator) ParseRequest(c *fiber.Ctx) (*models.CompletionRequest, error) {
	var body anthropicRequest
	if err := c.BodyParser(&body); err != nil {
		return nil, fmt.Errorf("translator.anthropic.ParseRequest: %w", err)
	}

	thinkingBudget := 0
	if body.Thinking != nil && body.Thinking.Type == "enabled" {
		thinkingBudget = body.Thinking.BudgetTokens
	}

	return &models.CompletionRequest{
		Config: models.ModelConfig{
			Model:          body.Model,
			MaxTokens:      body.MaxTokens,
			Temperature:    body.Temperature,
			TopP:           body.TopP,
			Stream:         body.Stream,
			ThinkingBudget: thinkingBudget,
		},
		Payload: models.RequestPayload{
			Messages: body.Messages,
			System:   parseSystemField(body.System),
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

func (t *anthropicTranslator) StreamEvents(w *bufio.Writer, events <-chan models.StreamEvent) error {
	for event := range events {
		if err := marshalSSE(w, event); err != nil {
			return err
		}
		_ = w.Flush()
	}
	_, _ = fmt.Fprintf(w, "data: [DONE]\n\n")
	_ = w.Flush()
	return nil
}
