// Package resend implements mailer.Mailer using the Resend API.
// Nothing outside this package imports resend-go directly.
package resend

import (
	"context"
	"fmt"

	resendlib "github.com/resendlabs/resend-go"

	"github.com/thatsbass/veil/internal/mailer"
)

const fromAddress = "Veil <noreply@veil.dev>"

type provider struct {
	client *resendlib.Client
}

// NewProvider returns a mailer.Mailer backed by Resend.
func NewProvider(apiKey string) mailer.Mailer {
	return &provider{client: resendlib.NewClient(apiKey)}
}

func (p *provider) SendWelcome(ctx context.Context, to string) error {
	_, err := p.client.Emails.Send(&resendlib.SendEmailRequest{
		From:    fromAddress,
		To:      []string{to},
		Subject: "Welcome to Veil 👋",
		Html:    welcomeHTML(),
	})
	if err != nil {
		return fmt.Errorf("resend.SendWelcome: %w", err)
	}
	return nil
}

func (p *provider) SendQuotaWarning(ctx context.Context, to string, usedPercent int) error {
	_, err := p.client.Emails.Send(&resendlib.SendEmailRequest{
		From:    fromAddress,
		To:      []string{to},
		Subject: fmt.Sprintf("Veil — %d%% of your monthly quota used", usedPercent),
		Html:    quotaWarningHTML(usedPercent),
	})
	if err != nil {
		return fmt.Errorf("resend.SendQuotaWarning: %w", err)
	}
	return nil
}

func (p *provider) SendQuotaExceeded(ctx context.Context, to string) error {
	_, err := p.client.Emails.Send(&resendlib.SendEmailRequest{
		From:    fromAddress,
		To:      []string{to},
		Subject: "Veil — Monthly quota exceeded",
		Html:    quotaExceededHTML(),
	})
	if err != nil {
		return fmt.Errorf("resend.SendQuotaExceeded: %w", err)
	}
	return nil
}

func (p *provider) SendInvoice(ctx context.Context, to string, inv *mailer.Invoice) error {
	_, err := p.client.Emails.Send(&resendlib.SendEmailRequest{
		From:    fromAddress,
		To:      []string{to},
		Subject: fmt.Sprintf("Veil — Receipt for %s plan", inv.PlanName),
		Html:    invoiceHTML(inv),
	})
	if err != nil {
		return fmt.Errorf("resend.SendInvoice: %w", err)
	}
	return nil
}

// --- HTML templates (inline, MVP) ---

func welcomeHTML() string {
	return `<!DOCTYPE html><html><body style="font-family:sans-serif;max-width:560px;margin:40px auto;color:#111">
<h1 style="font-size:24px">Welcome to Veil</h1>
<p>Your account is ready. Generate your first API key from the dashboard and point your tools to <code>https://api.veil.dev</code>.</p>
<p>Same endpoints, up to 90% cheaper.</p>
<p style="color:#666;font-size:12px;margin-top:40px">Veil · support@veil.dev</p>
</body></html>`
}

func quotaWarningHTML(usedPercent int) string {
	return fmt.Sprintf(`<!DOCTYPE html><html><body style="font-family:sans-serif;max-width:560px;margin:40px auto;color:#111">
<h1 style="font-size:24px">%d%% of your quota used</h1>
<p>You have used <strong>%d%%</strong> of your monthly token quota.</p>
<p>Upgrade your plan to avoid interruptions: <a href="https://veil.dev/dashboard/billing">veil.dev/dashboard/billing</a></p>
<p style="color:#666;font-size:12px;margin-top:40px">Veil · support@veil.dev</p>
</body></html>`, usedPercent, usedPercent)
}

func quotaExceededHTML() string {
	return `<!DOCTYPE html><html><body style="font-family:sans-serif;max-width:560px;margin:40px auto;color:#111">
<h1 style="font-size:24px">Monthly quota exceeded</h1>
<p>Your API requests are paused until your quota resets or you upgrade your plan.</p>
<p><a href="https://veil.dev/dashboard/billing">Upgrade now</a> to restore access immediately.</p>
<p style="color:#666;font-size:12px;margin-top:40px">Veil · support@veil.dev</p>
</body></html>`
}

func invoiceHTML(inv *mailer.Invoice) string {
	return fmt.Sprintf(`<!DOCTYPE html><html><body style="font-family:sans-serif;max-width:560px;margin:40px auto;color:#111">
<h1 style="font-size:24px">Payment receipt</h1>
<table style="width:100%%;border-collapse:collapse;margin:24px 0">
  <tr><td style="padding:8px 0;color:#666">Invoice ID</td><td>%s</td></tr>
  <tr><td style="padding:8px 0;color:#666">Plan</td><td>%s</td></tr>
  <tr><td style="padding:8px 0;color:#666">Amount</td><td>$%.2f</td></tr>
  <tr><td style="padding:8px 0;color:#666">Period</td><td>%s – %s</td></tr>
  <tr><td style="padding:8px 0;color:#666">Paid on</td><td>%s</td></tr>
</table>
<p style="color:#666;font-size:12px;margin-top:40px">Veil · support@veil.dev</p>
</body></html>`,
		inv.ID,
		inv.PlanName,
		inv.AmountUSD,
		inv.PeriodStart.Format("Jan 2, 2006"),
		inv.PeriodEnd.Format("Jan 2, 2006"),
		inv.PaidAt.Format("Jan 2, 2006"),
	)
}
