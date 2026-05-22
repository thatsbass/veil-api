package keys

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"github.com/thatsbass/veil/internal/api"
	"github.com/thatsbass/veil/internal/auth"
	"github.com/thatsbass/veil/pkg/utils"
)

// Handler manages API key lifecycle for dashboard users.
type Handler struct {
	resolver api.UserResolver
	repo     KeyRepository
}

func NewHandler(resolver api.UserResolver, repo KeyRepository) *Handler {
	return &Handler{resolver: resolver, repo: repo}
}

// CreateAPIKey creates a new API key for the authenticated user.
//
// @Summary      Créer une clé API
// @Description  Génère une nouvelle clé vl_live_xxx. La valeur brute n'est retournée qu'une seule fois.
// @Tags         api-keys
// @Accept       json
// @Produce      json
// @Param        body  body      CreateKeyRequest   true  "Nom de la clé"
// @Success      201   {object}  CreateKeyResponse  "Clé créée — conserver la valeur 'key', elle ne sera plus visible"
// @Failure      400   {object}  ErrorResponse
// @Failure      401   {object}  ErrorResponse
// @Failure      500   {object}  ErrorResponse
// @Security     DashboardAuth
// @Router       /api/keys [post]
func (h *Handler) CreateAPIKey(c *fiber.Ctx) error {
	session := auth.DashboardUserFrom(c)
	if session == nil {
		return unauthorized(c)
	}

	var req CreateKeyRequest
	if err := c.BodyParser(&req); err != nil || req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "name is required"})
	}

	user, err := h.resolver.ResolveUser(c.Context(), session.Email)
	if err != nil {
		return internalError(c, err)
	}

	rawKey, err := utils.GenerateAPIKey()
	if err != nil {
		return internalError(c, err)
	}

	key, err := h.repo.Create(c.Context(), user.ID, req.Name, utils.HashAPIKey(rawKey))
	if err != nil {
		return internalError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(CreateKeyResponse{
		ID:        key.ID,
		Name:      key.Name,
		Key:       rawKey,
		CreatedAt: key.CreatedAt.Format("2006-01-02T15:04:05Z"),
	})
}

// ListAPIKeys returns all API keys for the authenticated user.
//
// @Summary      Lister les clés API
// @Description  Retourne toutes les clés actives. La valeur brute n'est jamais exposée ici.
// @Tags         api-keys
// @Produce      json
// @Success      200  {object}  ListKeysResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Security     DashboardAuth
// @Router       /api/keys [get]
func (h *Handler) ListAPIKeys(c *fiber.Ctx) error {
	session := auth.DashboardUserFrom(c)
	if session == nil {
		return unauthorized(c)
	}

	user, err := h.resolver.ResolveUser(c.Context(), session.Email)
	if err != nil {
		return internalError(c, err)
	}

	keys, err := h.repo.List(c.Context(), user.ID)
	if err != nil {
		return internalError(c, err)
	}

	items := make([]KeyItem, 0, len(keys))
	for _, k := range keys {
		item := KeyItem{ID: k.ID, Name: k.Name, CreatedAt: k.CreatedAt.Format("2006-01-02T15:04:05Z")}
		if k.LastUsed != nil {
			s := k.LastUsed.Format("2006-01-02T15:04:05Z")
			item.LastUsed = &s
		}
		items = append(items, item)
	}

	return c.JSON(ListKeysResponse{Keys: items})
}

// RevokeAPIKey permanently deletes a key.
//
// @Summary      Révoquer une clé API
// @Description  Supprime définitivement la clé. Les requêtes utilisant cette clé seront immédiatement rejetées.
// @Tags         api-keys
// @Produce      json
// @Param        id   path      string  true  "ID de la clé" example("550e8400-e29b-41d4-a716-446655440000")
// @Success      204  "Clé révoquée"
// @Failure      401  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Security     DashboardAuth
// @Router       /api/keys/{id} [delete]
func (h *Handler) RevokeAPIKey(c *fiber.Ctx) error {
	session := auth.DashboardUserFrom(c)
	if session == nil {
		return unauthorized(c)
	}

	user, err := h.resolver.ResolveUser(c.Context(), session.Email)
	if err != nil {
		return internalError(c, err)
	}

	if err := h.repo.Revoke(c.Context(), user.ID, c.Params("id")); err != nil {
		if errors.Is(err, ErrKeyNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{Error: "key not found"})
		}
		return internalError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// --- helpers ---

func unauthorized(c *fiber.Ctx) error {
	return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{Error: "unauthorized"})
}

func internalError(c *fiber.Ctx, _ error) error {
	return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: "internal error"})
}

// --- swagger types ---

// CreateKeyRequest is the body for POST /api/keys.
type CreateKeyRequest struct {
	Name string `json:"name" example:"cursor-work"`
}

// CreateKeyResponse is returned once on key creation. Save 'key' — it won't appear again.
type CreateKeyResponse struct {
	ID        string `json:"id"         example:"550e8400-e29b-41d4-a716-446655440000"`
	Name      string `json:"name"       example:"cursor-work"`
	Key       string `json:"key"        example:"vl_live_aB3xK9mN2pQrS7tU1vW4yZ8cD6eF0gH5jI"`
	CreatedAt string `json:"created_at" example:"2026-05-15T00:00:00Z"`
}

// KeyItem is a single entry in the list response (no raw key).
type KeyItem struct {
	ID        string  `json:"id"                   example:"550e8400-e29b-41d4-a716-446655440000"`
	Name      string  `json:"name"                 example:"cursor-work"`
	LastUsed  *string `json:"last_used,omitempty"  example:"2026-05-15T10:30:00Z"`
	CreatedAt string  `json:"created_at"           example:"2026-05-01T00:00:00Z"`
}

// ListKeysResponse is the response for GET /api/keys.
type ListKeysResponse struct {
	Keys []KeyItem `json:"keys"`
}

// ErrorResponse is the standard error body.
type ErrorResponse struct {
	Error string `json:"error" example:"key not found"`
}
