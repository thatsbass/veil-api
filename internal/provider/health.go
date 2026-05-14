package provider

import (
	"context"
	"fmt"
	"time"
)

// HealthChecker pings a provider and reports its latency.
type HealthChecker struct {
	provider Provider
}

func NewHealthChecker(p Provider) *HealthChecker {
	return &HealthChecker{provider: p}
}

type HealthResult struct {
	Provider  string        `json:"provider"`
	Healthy   bool          `json:"healthy"`
	LatencyMs int64         `json:"latency_ms"`
	Error     string        `json:"error,omitempty"`
}

func (h *HealthChecker) Check(ctx context.Context) HealthResult {
	start := time.Now()
	req := minimalPingRequest()
	_, err := h.provider.Complete(ctx, req)
	latency := time.Since(start).Milliseconds()

	result := HealthResult{
		Provider:  h.provider.Name(),
		Healthy:   err == nil,
		LatencyMs: latency,
	}
	if err != nil {
		result.Error = fmt.Sprintf("health.Check: %v", err)
	}
	return result
}
