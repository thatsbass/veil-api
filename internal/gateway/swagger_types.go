package gateway

// ---- Shared ----

// SwaggerMessage is a single chat message.
type SwaggerMessage struct {
	Role    string `json:"role" example:"user"`
	Content string `json:"content" example:"Hello, how can you help me?"`
}

// SwaggerContentBlock is one block in an Anthropic-style response.
type SwaggerContentBlock struct {
	Type string `json:"type" example:"text"`
	Text string `json:"text" example:"Hello! I can help you with many things."`
}

// SwaggerAnthropicUsage is the token usage in Anthropic format.
type SwaggerAnthropicUsage struct {
	InputTokens  int `json:"input_tokens" example:"12"`
	OutputTokens int `json:"output_tokens" example:"48"`
}

// SwaggerOpenAIUsage is the token usage in OpenAI format.
type SwaggerOpenAIUsage struct {
	PromptTokens     int `json:"prompt_tokens" example:"12"`
	CompletionTokens int `json:"completion_tokens" example:"48"`
	TotalTokens      int `json:"total_tokens" example:"60"`
}

// ErrorResponse is the standard error body returned on all failure codes.
type ErrorResponse struct {
	Error string `json:"error" example:"invalid api key"`
}

// ---- Anthropic Messages API ----

// AnthropicMessagesRequest is the body for POST /v1/messages.
type AnthropicMessagesRequest struct {
	Model     string           `json:"model" example:"deepseek-chat"`
	Messages  []SwaggerMessage `json:"messages"`
	System    string           `json:"system,omitempty" example:"You are a helpful assistant."`
	MaxTokens int              `json:"max_tokens" example:"1024"`
	Stream    bool             `json:"stream,omitempty" example:"false"`
}

// AnthropicMessagesResponse is the response from POST /v1/messages.
type AnthropicMessagesResponse struct {
	ID         string                `json:"id" example:"msg_01XFDUDYJgAACzvnptvVoYEL"`
	Type       string                `json:"type" example:"message"`
	Role       string                `json:"role" example:"assistant"`
	Model      string                `json:"model" example:"deepseek-chat"`
	Content    []SwaggerContentBlock `json:"content"`
	StopReason string                `json:"stop_reason,omitempty" example:"end_turn"`
	Usage      SwaggerAnthropicUsage `json:"usage"`
}

// ---- OpenAI Chat Completions API ----

// OpenAIChatRequest is the body for POST /v1/chat/completions.
type OpenAIChatRequest struct {
	Model     string           `json:"model" example:"deepseek-chat"`
	Messages  []SwaggerMessage `json:"messages"`
	MaxTokens int              `json:"max_tokens,omitempty" example:"1024"`
	Stream    bool             `json:"stream,omitempty" example:"false"`
}

// OpenAIChatChoice is one completion choice.
type OpenAIChatChoice struct {
	Index        int            `json:"index" example:"0"`
	Message      SwaggerMessage `json:"message"`
	FinishReason string         `json:"finish_reason" example:"stop"`
}

// OpenAIChatResponse is the response from POST /v1/chat/completions.
type OpenAIChatResponse struct {
	ID      string             `json:"id" example:"chatcmpl-abc123"`
	Model   string             `json:"model" example:"deepseek-chat"`
	Choices []OpenAIChatChoice `json:"choices"`
	Usage   SwaggerOpenAIUsage `json:"usage"`
}

// ---- OpenAI Responses API ----

// ResponsesAPIRequest is the body for POST /v1/responses.
type ResponsesAPIRequest struct {
	Model     string           `json:"model" example:"deepseek-chat"`
	Input     []SwaggerMessage `json:"input"`
	MaxTokens int              `json:"max_output_tokens,omitempty" example:"1024"`
	Stream    bool             `json:"stream,omitempty" example:"false"`
}

// ResponsesAPIOutputItem is one item in the output array.
type ResponsesAPIOutputItem struct {
	Type    string `json:"type" example:"message"`
	Role    string `json:"role" example:"assistant"`
	Content string `json:"content" example:"Hello! I can help you with many things."`
}

// ResponsesAPIResponse is the response from POST /v1/responses.
type ResponsesAPIResponse struct {
	ID     string                   `json:"id" example:"resp_abc123"`
	Model  string                   `json:"model" example:"deepseek-chat"`
	Output []ResponsesAPIOutputItem `json:"output"`
	Usage  SwaggerAnthropicUsage    `json:"usage"`
}

// ---- Models & Health ----

// ModelObject is one model entry.
type ModelObject struct {
	ID      string `json:"id" example:"deepseek-chat"`
	Object  string `json:"object" example:"model"`
	OwnedBy string `json:"owned_by" example:"veil"`
}

// ModelsListResponse is the response from GET /v1/models.
type ModelsListResponse struct {
	Object string        `json:"object" example:"list"`
	Data   []ModelObject `json:"data"`
}

// HealthResponse is the response from GET /health.
type HealthResponse struct {
	Status string `json:"status" example:"ok"`
}
