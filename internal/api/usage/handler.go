package usage

import (
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/thatsbass/veil/internal/api"
	"github.com/thatsbass/veil/internal/auth"
	"github.com/thatsbass/veil/internal/billing"
)

const historyDefaultLimit = 50

// Handler returns token usage statistics for dashboard users.
type Handler struct {
	resolver api.UserResolver
	billing  billing.BillingService
	repo     UsageRepository
}

func NewHandler(resolver api.UserResolver, bill billing.BillingService, repo UsageRepository) *Handler {
	return &Handler{resolver: resolver, billing: bill, repo: repo}
}

// GetCurrentUsage returns the current month's token consumption and quota.
//
// @Summary      Consommation du mois en cours
// @Description  Retourne les tokens consommés, le quota du plan, le pourcentage utilisé et la date de réinitialisation.
// @Tags         usage
// @Produce      json
// @Success      200  {object}  CurrentUsageResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Security     DashboardAuth
// @Router       /api/usage [get]
func (h *Handler) GetCurrentUsage(c *fiber.Ctx) error {
	session := auth.DashboardUserFrom(c)
	if session == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{Error: "unauthorized"})
	}

	user, err := h.resolver.ResolveUser(c.Context(), session.Email)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: "internal error"})
	}

	used, err := h.billing.GetMonthlyUsage(c.Context(), user.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: "internal error"})
	}

	quota := planQuota(string(user.Plan))
	percent := 0
	if quota > 0 {
		percent = int(used * 100 / quota)
	}

	return c.JSON(CurrentUsageResponse{
		UsedTokens:  used,
		QuotaTokens: quota,
		Percent:     percent,
		ResetsAt:    endOfMonth().Format(time.RFC3339),
	})
}

// GetUsageHistory returns the last 50 requests for the authenticated user.
//
// @Summary      Historique des requêtes
// @Description  Retourne les 50 dernières requêtes avec détail tokens, coût et latence.
// @Tags         usage
// @Produce      json
// @Success      200  {object}  UsageHistoryResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Security     DashboardAuth
// @Router       /api/usage/history [get]
func (h *Handler) GetUsageHistory(c *fiber.Ctx) error {
	session := auth.DashboardUserFrom(c)
	if session == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{Error: "unauthorized"})
	}

	user, err := h.resolver.ResolveUser(c.Context(), session.Email)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: "internal error"})
	}

	records, err := h.repo.GetHistory(c.Context(), user.ID, historyDefaultLimit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: "internal error"})
	}

	return c.JSON(UsageHistoryResponse{Requests: records})
}

func endOfMonth() time.Time {
	now := time.Now().UTC()
	return time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, time.UTC)
}

func planQuota(plan string) int64 {
	switch plan {
	case "starter":
		return 2_000_000
	case "pro":
		return 10_000_000
	case "team":
		return 50_000_000
	default:
		return 100_000
	}
}

// --- swagger types ---

// CurrentUsageResponse is the response for GET /api/usage.
type CurrentUsageResponse struct {
	UsedTokens  int64  `json:"used_tokens"  example:"12500"`
	QuotaTokens int64  `json:"quota_tokens" example:"100000"`
	Percent     int    `json:"percent"      example:"12"`
	ResetsAt    string `json:"resets_at"    example:"2026-06-01T00:00:00Z"`
}

// UsageHistoryResponse is the response for GET /api/usage/history.
type UsageHistoryResponse struct {
	Requests []*RequestRecord `json:"requests"`
}

// ErrorResponse is the standard error body.
type ErrorResponse struct {
	Error string `json:"error" example:"unauthorized"`
}
