package billing

import (
	"fmt"
	"io"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	"github.com/stripe/stripe-go/v78/webhook"
)

// StripeHandler processes Stripe webhook events.
type StripeHandler struct {
	webhookSecret string
	log           zerolog.Logger
}

func NewStripeHandler(webhookSecret string, log zerolog.Logger) *StripeHandler {
	return &StripeHandler{webhookSecret: webhookSecret, log: log}
}

func (h *StripeHandler) HandleWebhook(c *fiber.Ctx) error {
	payload, err := io.ReadAll(c.Request().BodyStream())
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "failed to read body"})
	}

	sig := c.Get("Stripe-Signature")
	event, err := webhook.ConstructEvent(payload, sig, h.webhookSecret)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid signature"})
	}

	h.log.Info().Str("type", string(event.Type)).Msg("stripe webhook received")

	switch event.Type {
	case "customer.subscription.updated", "customer.subscription.deleted":
		if err := h.handleSubscriptionChange(event); err != nil {
			h.log.Error().Err(err).Msg("stripe: failed to handle subscription change")
		}
	case "invoice.payment_succeeded":
		h.log.Info().Msg("stripe: payment succeeded")
	}

	return c.SendStatus(fiber.StatusOK)
}

func (h *StripeHandler) handleSubscriptionChange(event any) error {
	// Phase 2: update user plan in DB based on subscription status
	return fmt.Errorf("billing.StripeHandler.handleSubscriptionChange: not implemented")
}
