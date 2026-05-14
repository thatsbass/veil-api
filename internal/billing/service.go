package billing

import (
	"context"

	"github.com/thatsbass/veil/pkg/models"
)

// BillingService records token usage and manages quota counters.
type BillingService interface {
	RecordUsage(userID string, usage models.UsageInfo)
	GetMonthlyUsage(ctx context.Context, userID string) (int64, error)
}

type billingService struct {
	meter  *Meter
	quota  *QuotaManager
}

func NewBillingService(meter *Meter, quota *QuotaManager) BillingService {
	return &billingService{meter: meter, quota: quota}
}

func (s *billingService) RecordUsage(userID string, usage models.UsageInfo) {
	total := int64(usage.InputTokens + usage.OutputTokens)
	s.meter.IncrementAsync(userID, total)
}

func (s *billingService) GetMonthlyUsage(ctx context.Context, userID string) (int64, error) {
	return s.quota.GetUsage(ctx, userID)
}
