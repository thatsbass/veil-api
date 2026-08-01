package v1handler

import (
	"bufio"
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	apibilling "github.com/thatsbass/veil/internal/api/billing"
	"github.com/thatsbass/veil/internal/auth"
	"github.com/thatsbass/veil/internal/billing"
	"github.com/thatsbass/veil/pkg/models"
)

// Handler serves /v1/usage, /v1/billing/plan, and /v1/logs — all protected by APIKeyMiddleware.
type Handler struct {
	billing billing.BillingService
	plans   apibilling.PlanRepository
	rdb     *redis.Client
	log     zerolog.Logger
}

func NewHandler(
	billingSvc billing.BillingService,
	plans apibilling.PlanRepository,
	rdb *redis.Client,
	log zerolog.Logger,
) *Handler {
	return &Handler{billing: billingSvc, plans: plans, rdb: rdb, log: log}
}

// GetUsage returns the authenticated user's current monthly token usage.
//
// @Summary      Token usage (CLI)
// @Tags         cli
// @Produce      json
// @Success      200  {object}  UsageResponse
// @Failure      401  {object}  map[string]string
// @Security     VeilAPIKey
// @Router       /v1/usage [get]
func (h *Handler) GetUsage(c *fiber.Ctx) error {
	user := auth.APIKeyUserFrom(c)

	used, err := h.billing.GetMonthlyUsage(c.Context(), user.ID)
	if err != nil {
		h.log.Error().Err(err).Str("user_id", user.ID).Msg("v1handler: get monthly usage")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}

	quota := models.PlanQuota(user.Plan)
	percent := 0
	if quota > 0 {
		percent = int(used * 100 / quota)
	}

	return c.JSON(fiber.Map{
		"used_tokens":  used,
		"quota_tokens": quota,
		"percent":      percent,
		"resets_at":    endOfMonth().Format("2006-01-02"),
	})
}

// GetBillingPlan returns the authenticated user's current plan details.
//
// @Summary      Billing plan (CLI)
// @Tags         cli
// @Produce      json
// @Success      200  {object}  apibilling.PlanRecord
// @Failure      401  {object}  map[string]string
// @Security     VeilAPIKey
// @Router       /v1/billing/plan [get]
func (h *Handler) GetBillingPlan(c *fiber.Ctx) error {
	user := auth.APIKeyUserFrom(c)

	plan, err := h.plans.GetUserPlan(c.Context(), user.ID)
	if err != nil {
		h.log.Error().Err(err).Str("user_id", user.ID).Msg("v1handler: get billing plan")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}

	return c.JSON(plan)
}

// StreamLogs streams live request events for the authenticated user over SSE.
//
// @Summary      Live log stream (CLI)
// @Tags         cli
// @Produce      text/event-stream
// @Success      200
// @Failure      401  {object}  map[string]string
// @Security     VeilAPIKey
// @Router       /v1/logs [get]
func (h *Handler) StreamLogs(c *fiber.Ctx) error {
	user := auth.APIKeyUserFrom(c)
	userID := user.ID

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")

	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		sub := h.rdb.Subscribe(ctx, "logs:"+userID)
		defer sub.Close()

		for msg := range sub.Channel() {
			if _, err := fmt.Fprintf(w, "data: %s\n\n", msg.Payload); err != nil {
				return
			}
			if err := w.Flush(); err != nil {
				return
			}
		}
	})
	return nil
}

func endOfMonth() time.Time {
	now := time.Now().UTC()
	return time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, time.UTC)
}
