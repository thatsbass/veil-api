package translator

import (
	"bufio"
	"encoding/json"
	"fmt"
	"strings"

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
	Model        string            `json:"model"`
	Input        json.RawMessage   `json:"input"`
	Instructions string            `json:"instructions,omitempty"`
	MaxTokens    int               `json:"max_output_tokens,omitempty"`
	Temperature  *float64          `json:"temperature,omitempty"`
	Stream       bool              `json:"stream,omitempty"`
	Tools        []json.RawMessage `json:"tools,omitempty"`
}

func (t *responsesTranslator) ParseRequest(c *fiber.Ctx) (*models.CompletionRequest, error) {
	var body responsesRequest
	if err := c.BodyParser(&body); err != nil {
		return nil, fmt.Errorf("translator.responses.ParseRequest: %w", err)
	}

	messages, err := parseResponsesInput(body.Input)
	if err != nil {
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
			Messages: messages,
			System:   body.Instructions,
			Tools:    parseResponsesTools(body.Tools),
		},
	}, nil
}

// responsesInputItem is one element of the Responses API input array.
type responsesInputItem struct {
	Type      string          `json:"type"`
	Role      string          `json:"role"`
	Content   json.RawMessage `json:"content"`
	CallID    string          `json:"call_id"`
	Name      string          `json:"name"`
	Arguments string          `json:"arguments"`
	Output    string          `json:"output"`
}

// parseResponsesInput handles input as a plain string, a message array, or an
// item array containing function_call / function_call_output entries.
func parseResponsesInput(raw json.RawMessage) ([]models.Message, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("input is required")
	}

	// Plain string → single user message.
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return []models.Message{{Role: "user", Content: s}}, nil
	}

	var items []responsesInputItem
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, fmt.Errorf("input must be a string or array of items")
	}

	var messages []models.Message
	i := 0
	for i < len(items) {
		item := items[i]
		switch item.Type {
		case "message":
			messages = append(messages, models.Message{
				Role:    item.Role,
				Content: responsesContentToAnthropic(item.Content),
			})
			i++

		case "function_call":
			// Group consecutive function_calls into one assistant message with
			// multiple tool_use blocks (required by the Anthropic protocol).
			var blocks []map[string]any
			for i < len(items) && items[i].Type == "function_call" {
				fc := items[i]
				var input any = map[string]any{}
				if fc.Arguments != "" {
					_ = json.Unmarshal([]byte(fc.Arguments), &input)
				}
				blocks = append(blocks, map[string]any{
					"type":  "tool_use",
					"id":    fc.CallID,
					"name":  fc.Name,
					"input": input,
				})
				i++
			}
			messages = append(messages, models.Message{Role: "assistant", Content: blocks})

		case "function_call_output":
			// Group consecutive function_call_outputs into one user message with
			// multiple tool_result blocks.
			var blocks []map[string]any
			for i < len(items) && items[i].Type == "function_call_output" {
				fo := items[i]
				blocks = append(blocks, map[string]any{
					"type":        "tool_result",
					"tool_use_id": fo.CallID,
					"content":     fo.Output,
				})
				i++
			}
			messages = append(messages, models.Message{Role: "user", Content: blocks})

		default:
			i++
		}
	}
	return messages, nil
}

// responsesContentToAnthropic converts Responses API content to Anthropic format.
// OpenAI uses "input_text"/"output_text" types; Anthropic uses "text".
func responsesContentToAnthropic(raw json.RawMessage) any {
	if raw == nil {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &parts); err != nil {
		return string(raw)
	}
	var blocks []map[string]string
	for _, p := range parts {
		t := p.Type
		if t == "input_text" || t == "output_text" {
			t = "text"
		}
		if t == "text" {
			blocks = append(blocks, map[string]string{"type": t, "text": p.Text})
		}
	}
	if len(blocks) > 0 {
		return blocks
	}
	return string(raw)
}

// parseResponsesTools converts OpenAI Responses API tools to Anthropic tool format.
// Non-function tools (e.g. "web_search_preview") are skipped.
func parseResponsesTools(rawTools []json.RawMessage) []models.Tool {
	if len(rawTools) == 0 {
		return nil
	}
	var tools []models.Tool
	for _, raw := range rawTools {
		var t struct {
			Type        string `json:"type"`
			Name        string `json:"name"`
			Description string `json:"description"`
			Parameters  any    `json:"parameters"`
		}
		if err := json.Unmarshal(raw, &t); err != nil || t.Type != "function" {
			continue
		}
		tools = append(tools, models.Tool{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: t.Parameters,
		})
	}
	return tools
}

func (t *responsesTranslator) WriteResponse(c *fiber.Ctx, resp *models.CompletionResponse) error {
	var output []fiber.Map
	for _, block := range resp.Content {
		switch block.Type {
		case "text":
			output = append(output, fiber.Map{
				"type": "message",
				"role": "assistant",
				"content": []fiber.Map{{
					"type": "output_text",
					"text": block.Text,
				}},
			})
		case "tool_use":
			args := "{}"
			if block.Input != nil {
				args = string(block.Input)
			}
			output = append(output, fiber.Map{
				"type":      "function_call",
				"id":        block.ID,
				"call_id":   block.ID,
				"name":      block.Name,
				"arguments": args,
			})
		}
	}
	if output == nil {
		output = []fiber.Map{}
	}
	return c.JSON(fiber.Map{
		"id":    resp.ID,
		"model": resp.Model,
		"output": output,
		"usage": fiber.Map{
			"input_tokens":  resp.Usage.InputTokens,
			"output_tokens": resp.Usage.OutputTokens,
		},
	})
}

// blockState tracks per-content-block state during streaming so that
// input_json_delta fragments can be accumulated and emitted correctly.
type blockState struct {
	isTool bool
	id     string
	name   string
	args   strings.Builder
}

// StreamEvents converts Anthropic SSE events to OpenAI Responses API streaming
// format. Tool calls are tracked across content_block_start/delta/stop triples.
func (t *responsesTranslator) StreamEvents(w *bufio.Writer, events <-chan models.StreamEvent) error {
	blocks := map[int]*blockState{}

	for event := range events {
		switch event.Type {

		case "content_block_start":
			var cb struct {
				Type string `json:"type"`
				ID   string `json:"id"`
				Name string `json:"name"`
			}
			if event.ContentBlock == nil {
				continue
			}
			if err := json.Unmarshal(event.ContentBlock, &cb); err != nil {
				continue
			}
			bs := &blockState{isTool: cb.Type == "tool_use", id: cb.ID, name: cb.Name}
			blocks[event.Index] = bs
			if bs.isTool {
				_ = marshalSSE(w, fiber.Map{
					"type":         "response.output_item.added",
					"output_index": event.Index,
					"item": fiber.Map{
						"type":      "function_call",
						"id":        cb.ID,
						"call_id":   cb.ID,
						"name":      cb.Name,
						"arguments": "",
					},
				})
				_ = w.Flush()
			}

		case "content_block_delta":
			if event.Delta == nil {
				continue
			}
			var delta struct {
				Type        string `json:"type"`
				Text        string `json:"text"`
				PartialJSON string `json:"partial_json"`
			}
			if err := json.Unmarshal(event.Delta, &delta); err != nil {
				continue
			}
			switch delta.Type {
			case "text_delta":
				_ = marshalSSE(w, fiber.Map{
					"type":  "response.output_text.delta",
					"delta": delta.Text,
				})
				_ = w.Flush()
			case "input_json_delta":
				if bs, ok := blocks[event.Index]; ok && bs.isTool {
					bs.args.WriteString(delta.PartialJSON)
					_ = marshalSSE(w, fiber.Map{
						"type":         "response.function_call_arguments.delta",
						"output_index": event.Index,
						"call_id":      bs.id,
						"delta":        delta.PartialJSON,
					})
					_ = w.Flush()
				}
			}

		case "content_block_stop":
			if bs, ok := blocks[event.Index]; ok && bs.isTool {
				_ = marshalSSE(w, fiber.Map{
					"type":         "response.function_call_arguments.done",
					"output_index": event.Index,
					"call_id":      bs.id,
					"arguments":    bs.args.String(),
				})
				_ = w.Flush()
				delete(blocks, event.Index)
			}

		case "message_stop":
			_ = marshalSSE(w, fiber.Map{"type": "response.completed"})
			_ = w.Flush()
		}
	}
	_, _ = fmt.Fprintf(w, "data: [DONE]\n\n")
	_ = w.Flush()
	return nil
}

// marshalSSE writes a value as "data: <json>\n\n" to w.
func marshalSSE(w interface{ Write([]byte) (int, error) }, v any) error {
	raw, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshalSSE: %w", err)
	}
	_, err = fmt.Fprintf(w, "data: %s\n\n", raw)
	return err
}
