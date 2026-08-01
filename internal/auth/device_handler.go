package auth

import (
	"context"
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/thatsbass/veil/pkg/models"
	"github.com/thatsbass/veil/pkg/utils"
)

// userResolver bridges Clerk email → Veil UUID.
// Local interface avoids circular import with internal/api.
type userResolver interface {
	ResolveUser(ctx context.Context, email string) (*models.User, error)
}

// keyCreator stores a new API key and returns the stored record.
// Local interface avoids circular import with internal/api/keys.
type keyCreator interface {
	Create(ctx context.Context, userID, name, keyHash string) (*models.APIKey, error)
}

// DeviceHandler implements the OAuth 2.0 Device Authorization Grant (RFC 8628).
type DeviceHandler struct {
	store    DeviceStore
	users    userResolver
	keys     keyCreator
	mailer   deviceMailer
	baseURL  string
	log      zerolog.Logger
}

// deviceMailer is a minimal mailer interface to avoid importing internal/mailer.
type deviceMailer interface {
	SendWelcome(ctx context.Context, to string) error
}

// NewDeviceHandler creates a DeviceHandler.
func NewDeviceHandler(
	store DeviceStore,
	users userResolver,
	keys keyCreator,
	mailer deviceMailer,
	baseURL string,
	log zerolog.Logger,
) *DeviceHandler {
	return &DeviceHandler{
		store:   store,
		users:   users,
		keys:    keys,
		mailer:  mailer,
		baseURL: baseURL,
		log:     log,
	}
}

// InitiateDeviceAuth starts a new device auth session.
//
// @Summary      Initier l'authentification CLI
// @Description  Démarre le Device Authorization Flow (RFC 8628). Le CLI affiche user_code et ouvre verification_url dans le navigateur.
// @Tags         auth
// @Produce      json
// @Success      200  {object}  DeviceAuthResponse
// @Failure      500  {object}  DeviceErrorResponse
// @Router       /auth/device [post]
func (h *DeviceHandler) InitiateDeviceAuth(c *fiber.Ctx) error {
	sess, err := h.store.Create(c.Context())
	if err != nil {
		h.log.Error().Err(err).Msg("device.InitiateDeviceAuth: failed to create session")
		return c.Status(fiber.StatusInternalServerError).JSON(DeviceErrorResponse{Error: "internal error"})
	}
	return c.JSON(DeviceAuthResponse{
		DeviceCode:      sess.DeviceCode,
		UserCode:        sess.UserCode,
		VerificationURL: h.baseURL + "/activate?user_code=" + sess.UserCode,
		ExpiresIn:       int(deviceSessionTTL.Seconds()),
		Interval:        3,
	})
}

// GetPendingDevice returns session info for a user_code so the dashboard can
// display the activation screen.
func (h *DeviceHandler) GetPendingDevice(c *fiber.Ctx) error {
	userCode := c.Query("user_code")
	if userCode == "" {
		return c.Status(fiber.StatusBadRequest).JSON(DeviceErrorResponse{Error: "user_code is required"})
	}

	sess, err := h.store.GetByUserCode(c.Context(), userCode)
	if err != nil {
		if errors.Is(err, ErrDeviceSessionNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(DeviceErrorResponse{Error: "invalid or expired code"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(DeviceErrorResponse{Error: "internal error"})
	}

	if time.Now().After(sess.ExpiresAt) {
		return c.Status(fiber.StatusGone).JSON(DeviceErrorResponse{Error: "code expired"})
	}

	return c.JSON(fiber.Map{
		"user_code":  sess.UserCode,
		"expires_at": sess.ExpiresAt,
	})
}

// PollToken allows the CLI to poll for the API key after user confirms.
//
// @Summary      Sonder le statut de l'authentification CLI
// @Description  Le CLI appelle cet endpoint toutes les 3 secondes jusqu'à obtenir le statut "authorized".
// @Tags         auth
// @Produce      json
// @Param        device_code  query     string  true  "Device code reçu lors de l'initiation"
// @Success      200          {object}  TokenResponse
// @Failure      400          {object}  DeviceErrorResponse
// @Router       /auth/device/token [get]
func (h *DeviceHandler) PollToken(c *fiber.Ctx) error {
	deviceCode := c.Query("device_code")
	if deviceCode == "" {
		return c.Status(fiber.StatusBadRequest).JSON(DeviceErrorResponse{Error: "device_code is required"})
	}

	sess, err := h.store.GetByDeviceCode(c.Context(), deviceCode)
	if err != nil {
		if errors.Is(err, ErrDeviceSessionNotFound) {
			return c.JSON(TokenResponse{Status: "expired"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(DeviceErrorResponse{Error: "internal error"})
	}

	if time.Now().After(sess.ExpiresAt) {
		return c.JSON(TokenResponse{Status: "expired"})
	}

	if !sess.Confirmed {
		return c.JSON(TokenResponse{Status: "authorization_pending"})
	}

	return c.JSON(TokenResponse{Status: "authorized", APIKey: sess.APIKey})
}

// ConfirmDevice is called by the dashboard after the user approves the CLI session.
//
// @Summary      Confirmer l'authentification CLI
// @Description  Le dashboard appelle cet endpoint après que l'utilisateur a approuvé la session CLI. Génère une clé API et la stocke dans la session Redis.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      ConfirmDeviceRequest  true  "Code utilisateur affiché dans le CLI"
// @Success      204   "Session confirmée"
// @Failure      400   {object}  DeviceErrorResponse
// @Failure      401   {object}  DeviceErrorResponse
// @Failure      404   {object}  DeviceErrorResponse
// @Failure      500   {object}  DeviceErrorResponse
// @Security     DashboardAuth
// @Router       /auth/device/confirm [post]
func (h *DeviceHandler) ConfirmDevice(c *fiber.Ctx) error {
	session := DashboardUserFrom(c)
	if session == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(DeviceErrorResponse{Error: "unauthorized"})
	}

	var req ConfirmDeviceRequest
	if err := c.BodyParser(&req); err != nil || req.UserCode == "" {
		return c.Status(fiber.StatusBadRequest).JSON(DeviceErrorResponse{Error: "user_code is required"})
	}

	user, err := h.users.ResolveUser(c.Context(), session.Email)
	if err != nil {
		h.log.Error().Err(err).Str("email", session.Email).Msg("device.ConfirmDevice: user not found")
		return c.Status(fiber.StatusInternalServerError).JSON(DeviceErrorResponse{Error: "user not found"})
	}

	rawKey, err := utils.GenerateAPIKey()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(DeviceErrorResponse{Error: "internal error"})
	}

	if _, err := h.keys.Create(c.Context(), user.ID, "veil-cli", utils.HashAPIKey(rawKey)); err != nil {
		h.log.Error().Err(err).Msg("device.ConfirmDevice: failed to create key")
		return c.Status(fiber.StatusInternalServerError).JSON(DeviceErrorResponse{Error: "internal error"})
	}

	if err := h.store.Confirm(c.Context(), req.UserCode, user.ID, rawKey); err != nil {
		if errors.Is(err, ErrDeviceSessionNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(DeviceErrorResponse{Error: "code not found or expired"})
		}
		if errors.Is(err, ErrDeviceSessionExpired) {
			return c.Status(fiber.StatusBadRequest).JSON(DeviceErrorResponse{Error: "code expired"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(DeviceErrorResponse{Error: "internal error"})
	}

	go func() {
		if err := h.mailer.SendWelcome(context.Background(), session.Email); err != nil {
			h.log.Warn().Err(err).Str("email", session.Email).Msg("device.ConfirmDevice: welcome email failed")
		}
	}()

	return c.SendStatus(fiber.StatusNoContent)
}

// --- swagger types ---

// DeviceAuthResponse is the response from POST /auth/device.
type DeviceAuthResponse struct {
	DeviceCode      string `json:"device_code"       example:"aB3xK9mN2pQrS7tU1vW4yZ8cD6eF0gH5jIkL"`
	UserCode        string `json:"user_code"         example:"ABCD-EFGH"`
	VerificationURL string `json:"verification_url"  example:"https://app.veil.dev/auth/device/activate"`
	ExpiresIn       int    `json:"expires_in"        example:"600"`
	Interval        int    `json:"interval"          example:"3"`
}

// TokenResponse is the response from GET /auth/device/token.
type TokenResponse struct {
	Status string `json:"status"              example:"authorized"`
	APIKey string `json:"api_key,omitempty"   example:"vl_live_aB3xK9mN2pQrS7tU1vW4yZ8cD6eF0gH5"`
}

// ConfirmDeviceRequest is the body for POST /auth/device/confirm.
type ConfirmDeviceRequest struct {
	UserCode string `json:"user_code" example:"ABCD-EFGH"`
}

// DeviceErrorResponse is the standard error for device auth endpoints.
type DeviceErrorResponse struct {
	Error string `json:"error" example:"code expired"`
}
