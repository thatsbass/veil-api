package models

import "time"

type RequestFormat string

const (
	FormatAnthropic  RequestFormat = "anthropic"
	FormatOpenAI     RequestFormat = "openai"
	FormatResponses  RequestFormat = "responses"
)

type Message struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type Tool struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	InputSchema any    `json:"input_schema,omitempty"`
}

type RequestMeta struct {
	UserID    string
	RequestID string
	Format    RequestFormat
	Timestamp time.Time
}

type ModelConfig struct {
	Model       string   `json:"model"`
	MaxTokens   int      `json:"max_tokens,omitempty"`
	Temperature *float64 `json:"temperature,omitempty"`
	TopP        *float64 `json:"top_p,omitempty"`
	Stream      bool     `json:"stream,omitempty"`
}

type RequestPayload struct {
	Messages []Message `json:"messages"`
	System   string    `json:"system,omitempty"`
	Tools    []Tool    `json:"tools,omitempty"`
}

type CompletionRequest struct {
	Meta    RequestMeta
	Config  ModelConfig
	Payload RequestPayload
}
