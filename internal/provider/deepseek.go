package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/thatsbass/veil/pkg/models"
)

const (
	deepSeekBaseURL          = "https://api.deepseek.com/anthropic"
	deepSeekAnthropicVersion = "2023-06-01"
	deepSeekDefaultModel     = "deepseek-v4-pro"
	deepSeekDefaultMaxTokens = 8192
	deepSeekRequestTimeout   = 90 * time.Second
	// budget_tokens > reasoningMaxThreshold → "max" effort (~64K tokens of reasoning).
	// budget_tokens > 0 → "high" effort (~10K tokens of reasoning).
	reasoningMaxThreshold = 32_000
)

type deepSeekProvider struct {
	client *resty.Client
}

// NewDeepSeek creates a DeepSeek provider using the Anthropic-compatible endpoint.
func NewDeepSeek(apiKey string) Provider {
	client := resty.New().
		SetBaseURL(deepSeekBaseURL).
		SetHeader("Authorization", "Bearer "+apiKey).
		SetHeader("Content-Type", "application/json").
		SetHeader("anthropic-version", deepSeekAnthropicVersion).
		SetTimeout(deepSeekRequestTimeout)

	return &deepSeekProvider{client: client}
}

func (p *deepSeekProvider) Name() string { return "deepseek" }

func (p *deepSeekProvider) Health(ctx context.Context) HealthResult {
	start := time.Now()
	_, err := p.Complete(ctx, minimalPingRequest())
	latency := time.Since(start).Milliseconds()

	result := HealthResult{Provider: p.Name(), Healthy: err == nil, LatencyMs: latency}
	if err != nil {
		result.Error = fmt.Sprintf("provider.deepseek.Health: %v", err)
	}
	return result
}

func (p *deepSeekProvider) Complete(ctx context.Context, req *models.CompletionRequest) (*models.CompletionResponse, error) {
	body := buildRequest(req, false)

	resp, err := p.client.R().
		SetContext(ctx).
		SetBody(body).
		Post("/v1/messages")
	if err != nil {
		return nil, fmt.Errorf("provider.deepseek.Complete: %w", err)
	}
	if resp.IsError() {
		return nil, newProviderError(resp.StatusCode(), resp.String())
	}

	return parseResponse(resp.Body())
}

func (p *deepSeekProvider) CompleteStream(ctx context.Context, req *models.CompletionRequest, out chan<- models.StreamEvent) error {
	body := buildRequest(req, true)

	resp, err := p.client.R().
		SetContext(ctx).
		SetBody(body).
		SetDoNotParseResponse(true).
		Post("/v1/messages")
	if err != nil {
		return fmt.Errorf("provider.deepseek.CompleteStream: %w", err)
	}
	defer resp.RawBody().Close()

	// 64KB initial buffer, 4MB max — prevents silent truncation on large events
	// (moon-bridge uses same values in their SSE scanner).
	scanner := bufio.NewScanner(resp.RawBody())
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		payload := strings.TrimPrefix(line, "data: ")
		if payload == "[DONE]" {
			break
		}
		var event models.StreamEvent
		if err := json.Unmarshal([]byte(payload), &event); err != nil {
			continue
		}
		out <- event
	}
	return scanner.Err()
}

// --- request building ---

type dsOutputConfig struct {
	Effort string `json:"effort"` // "high" | "max"
}

type dsRequest struct {
	Model        string          `json:"model"`
	Messages     []dsMessage     `json:"messages"`
	System       string          `json:"system,omitempty"`
	MaxTokens    int             `json:"max_tokens"`
	Stream       bool            `json:"stream,omitempty"`
	Temperature  *float64        `json:"temperature,omitempty"`
	TopP         *float64        `json:"top_p,omitempty"`
	OutputConfig *dsOutputConfig `json:"output_config,omitempty"`
	Tools        []models.Tool   `json:"tools,omitempty"`
}

type dsMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

// buildRequest converts the internal CompletionRequest to DeepSeek Anthropic format.
func buildRequest(req *models.CompletionRequest, stream bool) dsRequest {
	maxTokens := req.Config.MaxTokens
	if maxTokens == 0 {
		maxTokens = deepSeekDefaultMaxTokens
	}

	msgs := make([]dsMessage, 0, len(req.Payload.Messages))
	for _, m := range req.Payload.Messages {
		msgs = append(msgs, dsMessage{
			Role:    m.Role,
			Content: stripReasoningBlocks(m.Content),
		})
	}

	r := dsRequest{
		Model:     deepSeekDefaultModel,
		Messages:  msgs,
		System:    req.Payload.System,
		MaxTokens: maxTokens,
		Stream:    stream,
		Tools:     req.Payload.Tools,
	}

	// Map Claude's thinking.budget_tokens → DeepSeek output_config.effort.
	// When reasoning is active, DeepSeek rejects temperature/top_p — omit them.
	if effort := reasoningEffort(req.Config.ThinkingBudget); effort != "" {
		r.OutputConfig = &dsOutputConfig{Effort: effort}
	} else {
		r.Temperature = req.Config.Temperature
		r.TopP = req.Config.TopP
	}

	return r
}

// reasoningEffort maps Claude's budget_tokens to a DeepSeek effort level.
// Returns "" when reasoning is disabled (budget = 0).
func reasoningEffort(budgetTokens int) string {
	if budgetTokens <= 0 {
		return ""
	}
	if budgetTokens > reasoningMaxThreshold {
		return "max" // ~64K reasoning tokens
	}
	return "high" // ~10K reasoning tokens
}

// stripReasoningBlocks removes reasoning_content / thinking blocks from a
// message's content. DeepSeek adds these to its own responses; if a client
// echoes them back in a follow-up turn, DeepSeek returns 400.
func stripReasoningBlocks(content any) any {
	blocks, ok := content.([]any)
	if !ok {
		return content
	}
	filtered := make([]any, 0, len(blocks))
	for _, b := range blocks {
		block, ok := b.(map[string]any)
		if !ok {
			filtered = append(filtered, b)
			continue
		}
		t, _ := block["type"].(string)
		if t == "reasoning_content" || t == "thinking" {
			continue
		}
		filtered = append(filtered, block)
	}
	if len(filtered) == 0 {
		return ""
	}
	return filtered
}

// --- response parsing ---

type dsResponse struct {
	ID         string `json:"id"`
	Model      string `json:"model"`
	StopReason string `json:"stop_reason"`
	Content    []struct {
		Type  string          `json:"type"`
		Text  string          `json:"text"`
		ID    string          `json:"id"`
		Name  string          `json:"name"`
		Input json.RawMessage `json:"input"`
	} `json:"content"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

func parseResponse(raw []byte) (*models.CompletionResponse, error) {
	var ds dsResponse
	if err := json.NewDecoder(bytes.NewReader(raw)).Decode(&ds); err != nil {
		return nil, fmt.Errorf("provider.deepseek.parseResponse: %w", err)
	}

	var content []models.ContentBlock
	for _, b := range ds.Content {
		switch b.Type {
		case "text":
			content = append(content, models.ContentBlock{Type: "text", Text: b.Text})
		case "tool_use":
			content = append(content, models.ContentBlock{
				Type:  "tool_use",
				ID:    b.ID,
				Name:  b.Name,
				Input: b.Input,
			})
		}
	}
	if len(content) == 0 {
		return nil, fmt.Errorf("provider.deepseek.parseResponse: empty response")
	}

	return &models.CompletionResponse{
		ID:         ds.ID,
		Model:      ds.Model,
		Content:    content,
		StopReason: ds.StopReason,
		Usage: models.UsageInfo{
			InputTokens:  ds.Usage.InputTokens,
			OutputTokens: ds.Usage.OutputTokens,
		},
	}, nil
}

func minimalPingRequest() *models.CompletionRequest {
	return &models.CompletionRequest{
		Config: models.ModelConfig{MaxTokens: 1},
		Payload: models.RequestPayload{
			Messages: []models.Message{{Role: "user", Content: "ping"}},
		},
	}
}
