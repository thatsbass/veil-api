package billing

import (
	"github.com/gofiber/fiber/v2"

	"github.com/thatsbass/veil/internal/api"
	"github.com/thatsbass/veil/internal/auth"
)

var validPlans = map[string]bool{
	"starter": true,
	"pro":     true,
	"team":    true,
}

// Handler returns billing and plan information for dashboard users.
type Handler struct {
	resolver api.UserResolver
	plans    PlanRepository
}

func NewHandler(resolver api.UserResolver, plans PlanRepository) *Handler {
	return &Handler{resolver: resolver, plans: plans}
}

// GetCurrentPlan returns the user's active plan and its limits.
//
// @Summary      Plan actuel
// @Description  Retourne le plan souscrit, son prix, le quota mensuel de tokens et le nombre de clés API autorisées.
// @Tags         billing
// @Produce      json
// @Success      200  {object}  PlanRecord
// @Failure      401  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Security     DashboardAuth
// @Router       /api/billing/plan [get]
func (h *Handler) GetCurrentPlan(c *fiber.Ctx) error {
	session := auth.DashboardUserFrom(c)
	if session == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{Error: "unauthorized"})
	}

	user, err := h.resolver.ResolveUser(c.Context(), session.Email)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: "internal error"})
	}

	plan, err := h.plans.GetUserPlan(c.Context(), user.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: "internal error"})
	}

	return c.JSON(plan)
}

// UpgradePlan initiates a plan upgrade via Stripe Checkout.
//
// @Summary      Mettre à niveau le plan
// @Description  Génère une URL Stripe Checkout pour upgrader vers starter, pro ou team. Redirige l'utilisateur vers cette URL.
// @Tags         billing
// @Accept       json
// @Produce      json
// @Param        body  body      UpgradePlanRequest   true  "Plan cible"
// @Success      200   {object}  UpgradePlanResponse  "URL de paiement Stripe"
// @Failure      400   {object}  ErrorResponse
// @Failure      401   {object}  ErrorResponse
// @Failure      500   {object}  ErrorResponse
// @Security     DashboardAuth
// @Router       /api/billing/upgrade [post]
func (h *Handler) UpgradePlan(c *fiber.Ctx) error {
	session := auth.DashboardUserFrom(c)
	if session == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{Error: "unauthorized"})
	}

	var req UpgradePlanRequest
	if err := c.BodyParser(&req); err != nil || !validPlans[req.Plan] {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "plan must be one of: starter, pro, team"})
	}

	// Stripe Checkout session creation — Phase 2.
	return c.Status(fiber.StatusNotImplemented).JSON(ErrorResponse{Error: "billing upgrade coming soon"})
}

// --- swagger types ---

// UpgradePlanRequest is the body for POST /api/billing/upgrade.
type UpgradePlanRequest struct {
	Plan string `json:"plan" example:"starter"`
}

// UpgradePlanResponse contains the Stripe Checkout redirect URL.
type UpgradePlanResponse struct {
	CheckoutURL string `json:"checkout_url" example:"https://checkout.stripe.com/pay/cs_test_xxx"`
}

// ErrorResponse is the standard error body.
type ErrorResponse struct {
	Error string `json:"error" example:"unauthorized"`
}
