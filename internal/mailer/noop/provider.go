// Package noop provides a no-op mailer for tests and local development.
// Every method logs the intent and returns nil — no emails are sent.
package noop

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"

	"github.com/thatsbass/veil/internal/mailer"
)

type provider struct{}

// NewProvider returns a mailer.Mailer that logs instead of sending emails.
func NewProvider() mailer.Mailer {
	return &provider{}
}

func (p *provider) SendWelcome(_ context.Context, to string) error {
	log.Debug().Str("to", to).Msg("noop: SendWelcome")
	return nil
}

func (p *provider) SendQuotaWarning(_ context.Context, to string, usedPercent int) error {
	log.Debug().Str("to", to).Msg(fmt.Sprintf("noop: SendQuotaWarning %d%%", usedPercent))
	return nil
}

func (p *provider) SendQuotaExceeded(_ context.Context, to string) error {
	log.Debug().Str("to", to).Msg("noop: SendQuotaExceeded")
	return nil
}

func (p *provider) SendInvoice(_ context.Context, to string, inv *mailer.Invoice) error {
	log.Debug().Str("to", to).Str("invoice_id", inv.ID).Msg("noop: SendInvoice")
	return nil
}
