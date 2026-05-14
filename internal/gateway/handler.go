package gateway

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	"github.com/thatsbass/veil/internal/analytics"
	"github.com/thatsbass/veil/internal/billing"
	"github.com/thatsbass/veil/internal/provider"
	"github.com/thatsbass/veil/internal/translator"
	"github.com/thatsbass/veil/pkg/models"
)

// Handler routes inbound requests to the right translator and upstream provider.
type Handler struct {
	translators translators
	provider    provider.Provider
	billing     billing.BillingService
	analytics   analytics.AnalyticsService
	log         zerolog.Logger
}

func NewHandler(
	anthropic, openai, responses translator.Translator,
	prov provider.Provider,
	bill billing.BillingService,
	anal analytics.AnalyticsService,
	log zerolog.Logger,
) *Handler {
	return &Handler{
		translators: translators{
			anthropic: anthropic,
			openai:    openai,
			responses: responses,
		},
		provider:  prov,
		billing:   bill,
		analytics: anal,
		log:       log,
	}
}

// HandleCompletion serves POST /v1/messages, /v1/chat/completions, and /v1/responses.
func (h *Handler) HandleCompletion(c *fiber.Ctx) error {
	user := c.Locals("user").(*models.User)
	format := detectFormat(c)
	t := selectTranslator(format, &h.translators)

	req, err := t.ParseRequest(c)
	if err != nil {
		return h.respondError(c, err)
	}
	req.Meta = models.RequestMeta{
		UserID: user.ID,
		Format: format,
	}

	if req.Config.Stream {
		return h.handleStream(c, t, req, user)
	}
	return h.handleBlocking(c, t, req, user)
}

func (h *Handler) handleBlocking(c *fiber.Ctx, t translator.Translator, req *models.CompletionRequest, user *models.User) error {
	resp, err := h.provider.Complete(c.Context(), req)
	if err != nil {
		return h.respondError(c, err)
	}
	go h.recordUsage(req, resp, user)
	return t.WriteResponse(c, resp)
}

func (h *Handler) handleStream(c *fiber.Ctx, t translator.Translator, req *models.CompletionRequest, user *models.User) error {
	events := make(chan models.StreamEvent, 32)
	go func() {
		defer close(events)
		if err := h.provider.CompleteStream(c.Context(), req, events); err != nil {
			h.log.Error().Err(err).Str("user_id", user.ID).Msg("stream error")
		}
	}()
	return streamResponse(c, t, events)
}

func (h *Handler) recordUsage(req *models.CompletionRequest, resp *models.CompletionResponse, user *models.User) {
	h.billing.RecordUsage(req.Meta.UserID, resp.Usage)
	h.analytics.Track(analytics.Event{
		UserID:   user.ID,
		Provider: h.provider.Name(),
		Format:   string(req.Meta.Format),
		Usage:    resp.Usage,
	})
}

// HandleModels serves GET /v1/models for client compatibility.
func (h *Handler) HandleModels(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"object": "list",
		"data": []fiber.Map{
			{"id": "deepseek-chat", "object": "model", "owned_by": "veil"},
		},
	})
}

// HandleHealth serves GET /health.
func (h *Handler) HandleHealth(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"status": "ok"})
}

func (h *Handler) respondError(c *fiber.Ctx, err error) error {
	status, body := mapErrorToResponse(err)
	h.log.Error().Err(err).Int("status", status).Msg("request failed")
	return c.Status(status).JSON(body)
}

func mapErrorToResponse(err error) (int, fiber.Map) {
	switch {
	case errors.Is(err, fiber.ErrBadRequest):
		return fiber.StatusBadRequest, fiber.Map{"error": err.Error()}
	default:
		return fiber.StatusInternalServerError, fiber.Map{"error": "internal error"}
	}
}
