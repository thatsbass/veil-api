package models

type UsageInfo struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type CompletionResponse struct {
	ID      string         `json:"id"`
	Model   string         `json:"model"`
	Content []ContentBlock `json:"content"`
	Usage   UsageInfo      `json:"usage"`

	// StopReason is "end_turn", "max_tokens", "tool_use", etc.
	StopReason string `json:"stop_reason,omitempty"`
}

// StreamEvent represents a single SSE event chunk.
type StreamEvent struct {
	Type  string `json:"type"`
	Index int    `json:"index,omitempty"`
	Delta *struct {
		Type string `json:"type"`
		Text string `json:"text,omitempty"`
	} `json:"delta,omitempty"`
	Usage *UsageInfo `json:"usage,omitempty"`
}
