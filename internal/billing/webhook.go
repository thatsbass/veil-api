package billing

import (
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
)

// WebhookHandler handles inbound payment events via any PaymentProvider.
type WebhookHandler struct {
	provider PaymentProvider
	log      zerolog.Logger
}

func NewWebhookHandler(provider PaymentProvider, log zerolog.Logger) *WebhookHandler {
	return &WebhookHandler{provider: provider, log: log}
}

func (h *WebhookHandler) Handle(c *fiber.Ctx) error {
	sig := c.Get(h.provider.SignatureHeader())
	event, err := h.provider.VerifyWebhook(c.Body(), sig)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid webhook signature"})
	}

	h.log.Info().Str("type", string(event.Type)).Msg("payment webhook received")

	switch event.Type {
	case EventSubscriptionUpdated, EventSubscriptionDeleted:
		// Phase 2: update user plan in DB based on PaymentCustomerID.
		h.log.Warn().
			Str("customer_id", event.PaymentCustomerID).
			Str("plan_id", event.PlanID).
			Msg("payment: subscription change handler not yet implemented")
	case EventPaymentSucceeded:
		h.log.Info().Str("invoice_id", event.InvoiceID).Msg("payment: payment succeeded")
	}

	return c.SendStatus(fiber.StatusOK)
}
