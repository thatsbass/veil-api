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
	deepSeekBaseURL        = "https://api.deepseek.com/v1"
	deepSeekDefaultModel   = "deepseek-chat"
	deepSeekRequestTimeout = 60 * time.Second
)

type deepSeekProvider struct {
	client *resty.Client
}

// NewDeepSeek creates a DeepSeek provider client.
func NewDeepSeek(apiKey string) Provider {
	client := resty.New().
		SetBaseURL(deepSeekBaseURL).
		SetHeader("Authorization", "Bearer "+apiKey).
		SetHeader("Content-Type", "application/json").
		SetTimeout(deepSeekRequestTimeout)

	return &deepSeekProvider{client: client}
}

func (p *deepSeekProvider) Name() string { return "deepseek" }

func (p *deepSeekProvider) Complete(ctx context.Context, req *models.CompletionRequest) (*models.CompletionResponse, error) {
	body := buildDeepSeekRequest(req, false)

	resp, err := p.client.R().
		SetContext(ctx).
		SetBody(body).
		Post("/chat/completions")
	if err != nil {
		return nil, fmt.Errorf("provider.deepseek.Complete: %w", err)
	}
	if resp.IsError() {
		return nil, fmt.Errorf("provider.deepseek.Complete: upstream %d: %s", resp.StatusCode(), resp.String())
	}

	return parseDeepSeekResponse(resp.Body())
}

func (p *deepSeekProvider) CompleteStream(ctx context.Context, req *models.CompletionRequest, out chan<- models.StreamEvent) error {
	body := buildDeepSeekRequest(req, true)

	resp, err := p.client.R().
		SetContext(ctx).
		SetBody(body).
		SetDoNotParseResponse(true).
		Post("/chat/completions")
	if err != nil {
		return fmt.Errorf("provider.deepseek.CompleteStream: %w", err)
	}
	defer resp.RawBody().Close()

	scanner := bufio.NewScanner(resp.RawBody())
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

// --- internal helpers ---

type deepSeekMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type deepSeekRequest struct {
	Model       string            `json:"model"`
	Messages    []deepSeekMessage `json:"messages"`
	MaxTokens   int               `json:"max_tokens,omitempty"`
	Temperature *float64          `json:"temperature,omitempty"`
	TopP        *float64          `json:"top_p,omitempty"`
	Stream      bool              `json:"stream,omitempty"`
}

func buildDeepSeekRequest(req *models.CompletionRequest, stream bool) deepSeekRequest {
	model := req.Config.Model
	if model == "" {
		model = deepSeekDefaultModel
	}

	msgs := make([]deepSeekMessage, 0, len(req.Payload.Messages)+1)
	if req.Payload.System != "" {
		msgs = append(msgs, deepSeekMessage{Role: "system", Content: req.Payload.System})
	}
	for _, m := range req.Payload.Messages {
		content, _ := m.Content.(string)
		msgs = append(msgs, deepSeekMessage{Role: m.Role, Content: content})
	}

	return deepSeekRequest{
		Model:       model,
		Messages:    msgs,
		MaxTokens:   req.Config.MaxTokens,
		Temperature: req.Config.Temperature,
		TopP:        req.Config.TopP,
		Stream:      stream,
	}
}

type deepSeekResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

func parseDeepSeekResponse(raw []byte) (*models.CompletionResponse, error) {
	var ds deepSeekResponse
	if err := json.NewDecoder(bytes.NewReader(raw)).Decode(&ds); err != nil {
		return nil, fmt.Errorf("provider.deepseek.parseResponse: %w", err)
	}
	if len(ds.Choices) == 0 {
		return nil, fmt.Errorf("provider.deepseek.parseResponse: no choices in response")
	}

	return &models.CompletionResponse{
		ID:    ds.ID,
		Model: ds.Model,
		Content: []models.ContentBlock{
			{Type: "text", Text: ds.Choices[0].Message.Content},
		},
		StopReason: ds.Choices[0].FinishReason,
		Usage: models.UsageInfo{
			InputTokens:  ds.Usage.PromptTokens,
			OutputTokens: ds.Usage.CompletionTokens,
		},
	}, nil
}

func minimalPingRequest() *models.CompletionRequest {
	return &models.CompletionRequest{
		Config: models.ModelConfig{Model: deepSeekDefaultModel, MaxTokens: 1},
		Payload: models.RequestPayload{
			Messages: []models.Message{{Role: "user", Content: "ping"}},
		},
	}
}
