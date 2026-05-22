// Package stripe implements billing.PaymentProvider using Stripe webhooks.
// Nothing outside this package imports stripe-go directly.
package stripe

import (
	"fmt"

	stripelib "github.com/stripe/stripe-go/v78"
	"github.com/stripe/stripe-go/v78/webhook"

	"github.com/thatsbass/veil/internal/billing"
)

const signatureHeader = "Stripe-Signature"

type provider struct {
	webhookSecret string
}

// NewProvider returns a billing.PaymentProvider backed by Stripe.
func NewProvider(webhookSecret string) billing.PaymentProvider {
	return &provider{webhookSecret: webhookSecret}
}

func (p *provider) SignatureHeader() string {
	return signatureHeader
}

func (p *provider) VerifyWebhook(payload []byte, sig string) (*billing.PaymentEvent, error) {
	event, err := webhook.ConstructEvent(payload, sig, p.webhookSecret)
	if err != nil {
		return nil, fmt.Errorf("stripe.VerifyWebhook: %w", err)
	}
	return mapEvent(event), nil
}

func mapEvent(e stripelib.Event) *billing.PaymentEvent {
	ev := &billing.PaymentEvent{Type: mapEventType(string(e.Type))}
	if obj := e.Data.Object; obj != nil {
		if id, ok := obj["customer"].(string); ok {
			ev.PaymentCustomerID = id
		}
		if id, ok := obj["id"].(string); ok {
			ev.InvoiceID = id
		}
		if items, ok := obj["items"].(map[string]interface{}); ok {
			if data, ok := items["data"].([]interface{}); ok && len(data) > 0 {
				if item, ok := data[0].(map[string]interface{}); ok {
					if price, ok := item["price"].(map[string]interface{}); ok {
						ev.PlanID, _ = price["lookup_key"].(string)
					}
				}
			}
		}
	}
	return ev
}

func mapEventType(t string) billing.PaymentEventType {
	switch t {
	case "customer.subscription.updated":
		return billing.EventSubscriptionUpdated
	case "customer.subscription.deleted":
		return billing.EventSubscriptionDeleted
	case "invoice.payment_succeeded":
		return billing.EventPaymentSucceeded
	default:
		return billing.EventUnknown
	}
}
