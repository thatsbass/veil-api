package provider

import (
	"context"

	"github.com/thatsbass/veil/pkg/models"
)

// Provider forwards a completion request to an upstream LLM API.
type Provider interface {
	Complete(ctx context.Context, req *models.CompletionRequest) (*models.CompletionResponse, error)
	CompleteStream(ctx context.Context, req *models.CompletionRequest, out chan<- models.StreamEvent) error
	Name() string
}
