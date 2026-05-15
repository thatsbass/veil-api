package mailer

import (
	"context"
	"time"
)

// Mailer sends transactional emails. Implementations live in sub-packages.
type Mailer interface {
	SendWelcome(ctx context.Context, to string) error
	SendQuotaWarning(ctx context.Context, to string, usedPercent int) error
	SendQuotaExceeded(ctx context.Context, to string) error
	SendInvoice(ctx context.Context, to string, invoice *Invoice) error
}

// Invoice holds the data needed to render a payment receipt email.
type Invoice struct {
	ID          string
	PlanName    string
	AmountUSD   float64
	PeriodStart time.Time
	PeriodEnd   time.Time
	PaidAt      time.Time
}
