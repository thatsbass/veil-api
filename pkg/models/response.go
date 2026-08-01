package models

import "encoding/json"

type UsageInfo struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type ContentBlock struct {
	Type  string          `json:"type"`
	Text  string          `json:"text,omitempty"`
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
}

type CompletionResponse struct {
	ID      string         `json:"id"`
	Model   string         `json:"model"`
	Content []ContentBlock `json:"content"`
	Usage   UsageInfo      `json:"usage"`

	// StopReason is "end_turn", "max_tokens", "tool_use", etc.
	StopReason string `json:"stop_reason,omitempty"`
}

// StreamEvent is one SSE chunk from an Anthropic-format upstream.
// Delta, Message, and ContentBlock are kept as raw JSON so every field
// is forwarded to the client without loss, regardless of event type.
type StreamEvent struct {
	Type         string          `json:"type"`
	Index        int             `json:"index,omitempty"`
	Delta        json.RawMessage `json:"delta,omitempty"`
	Usage        *UsageInfo      `json:"usage,omitempty"`
	Message      json.RawMessage `json:"message,omitempty"`
	ContentBlock json.RawMessage `json:"content_block,omitempty"`
}
