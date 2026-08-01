package gateway

import (
	"context"
	"errors"
	"time"

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

// HandleMessages completes a chat request in Anthropic Messages format.
//
// @Summary      Messages (format Anthropic)
// @Description  Compatible Claude CLI / Claude SDK. Traduit automatiquement vers DeepSeek en arrière-plan.
// @Tags         completion
// @Accept       json
// @Produce      json
// @Param        body  body      AnthropicMessagesRequest   true  "Corps de la requête Anthropic"
// @Success      200   {object}  AnthropicMessagesResponse  "Réponse au format Anthropic Messages"
// @Failure      400   {object}  ErrorResponse
// @Failure      401   {object}  ErrorResponse
// @Failure      429   {object}  ErrorResponse
// @Failure      504   {object}  ErrorResponse
// @Failure      500   {object}  ErrorResponse
// @Security     VeilAPIKey
// @Router       /v1/messages [post]
func (h *Handler) HandleMessages(c *fiber.Ctx) error {
	return h.HandleCompletion(c)
}

// HandleChatCompletions completes a chat request in OpenAI Chat Completions format.
//
// @Summary      Chat Completions (format OpenAI)
// @Description  Compatible OpenAI SDK, Cursor, Aider, et tout client OpenAI-compatible. Traduit automatiquement vers DeepSeek.
// @Tags         completion
// @Accept       json
// @Produce      json
// @Param        body  body      OpenAIChatRequest   true  "Corps de la requête OpenAI"
// @Success      200   {object}  OpenAIChatResponse  "Réponse au format OpenAI Chat Completions"
// @Failure      400   {object}  ErrorResponse
// @Failure      401   {object}  ErrorResponse
// @Failure      429   {object}  ErrorResponse
// @Failure      504   {object}  ErrorResponse
// @Failure      500   {object}  ErrorResponse
// @Security     VeilAPIKey
// @Router       /v1/chat/completions [post]
func (h *Handler) HandleChatCompletions(c *fiber.Ctx) error {
	return h.HandleCompletion(c)
}

// HandleResponses completes a chat request in OpenAI Responses API format.
//
// @Summary      Responses API (format Codex)
// @Description  Compatible OpenAI Responses API (Codex CLI). Traduit automatiquement vers DeepSeek.
// @Tags         completion
// @Accept       json
// @Produce      json
// @Param        body  body      ResponsesAPIRequest   true  "Corps de la requête Responses API"
// @Success      200   {object}  ResponsesAPIResponse  "Réponse au format Responses API"
// @Failure      400   {object}  ErrorResponse
// @Failure      401   {object}  ErrorResponse
// @Failure      429   {object}  ErrorResponse
// @Failure      504   {object}  ErrorResponse
// @Failure      500   {object}  ErrorResponse
// @Security     VeilAPIKey
// @Router       /v1/responses [post]
func (h *Handler) HandleResponses(c *fiber.Ctx) error {
	return h.HandleCompletion(c)
}

// HandleCompletion is the shared implementation for all completion routes.
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
	start := time.Now()
	resp, err := h.provider.Complete(c.UserContext(), req)
	if err != nil {
		return h.respondError(c, err)
	}
	go h.recordUsage(req, resp, user, int(time.Since(start).Milliseconds()), "success")
	return t.WriteResponse(c, resp)
}

func (h *Handler) handleStream(c *fiber.Ctx, t translator.Translator, req *models.CompletionRequest, user *models.User) error {
	// Capture UserContext before spawning the goroutine — fasthttp recycles
	// c.Context() (*fasthttp.RequestCtx) once the handler returns, which causes
	// a nil-pointer panic in net/http's context propagation code.
	ctx := c.UserContext()
	events := make(chan models.StreamEvent, 32)
	go func() {
		defer close(events)
		if err := h.provider.CompleteStream(ctx, req, events); err != nil {
			h.log.Error().Err(err).Str("user_id", user.ID).Msg("stream error")
		}
	}()
	return streamResponse(c, t, events)
}

func (h *Handler) recordUsage(req *models.CompletionRequest, resp *models.CompletionResponse, user *models.User, latencyMS int, status string) {
	h.billing.RecordUsage(req.Meta.UserID, resp.Usage)
	h.analytics.Track(analytics.Event{
		UserID:    user.ID,
		Provider:  h.provider.Name(),
		Format:    string(req.Meta.Format),
		Usage:     resp.Usage,
		LatencyMS: latencyMS,
		Status:    status,
	})
}

// HandleModels lists available models for client compatibility.
//
// @Summary      Liste des modèles
// @Description  Retourne les modèles disponibles sur Veil. Compatible avec les clients qui appellent /v1/models au démarrage (Cursor, etc.).
// @Tags         models
// @Produce      json
// @Success      200  {object}  ModelsListResponse
// @Failure      401  {object}  ErrorResponse
// @Security     VeilAPIKey
// @Router       /v1/models [get]
func (h *Handler) HandleModels(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"object": "list",
		"data": []fiber.Map{
			{"id": "deepseek-chat", "object": "model", "owned_by": "veil"},
		},
	})
}

// HandleHealth returns the server liveness status.
//
// @Summary      Health check
// @Description  Endpoint de liveness utilisé par Dokploy / load-balancer. Retourne 200 si le serveur répond.
// @Tags         system
// @Produce      json
// @Success      200  {object}  HealthResponse
// @Router       /health [get]
func (h *Handler) HandleHealth(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"status": "ok"})
}

func (h *Handler) respondError(c *fiber.Ctx, err error) error {
	status, body := mapErrorToResponse(err)
	h.log.Error().Err(err).Int("status", status).Msg("request failed")
	return c.Status(status).JSON(body)
}

func mapErrorToResponse(err error) (int, fiber.Map) {
	var pe *provider.ProviderError
	if errors.As(err, &pe) {
		return pe.ClientStatus(), fiber.Map{
			"error": fiber.Map{
				"message": pe.Message,
				"type":    pe.Code,
				"code":    pe.Code,
			},
		}
	}
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return fiber.StatusGatewayTimeout, fiber.Map{"error": "upstream timeout"}
	case errors.Is(err, fiber.ErrBadRequest):
		return fiber.StatusBadRequest, fiber.Map{"error": err.Error()}
	default:
		return fiber.StatusInternalServerError, fiber.Map{"error": "internal error"}
	}
}
