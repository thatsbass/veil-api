package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/thatsbass/veil/pkg/models"
)

var (
	ErrMissingAPIKey    = errors.New("missing api key")
	ErrInvalidKeyFormat = errors.New("invalid api key format")
	ErrInvalidKeyLength = errors.New("invalid api key length")
	ErrInvalidAPIKey    = errors.New("invalid api key")
	ErrQuotaExceeded    = errors.New("monthly token quota exceeded")
)

const (
	keyPrefix    = "vl_live_"
	keyTotalLen  = 48
)

// AuthService validates API keys and enforces quota.
type AuthService interface {
	ValidateKey(ctx context.Context, rawKey string) (*models.User, error)
}

type authService struct {
	repo  Repository
	quota QuotaChecker
}

func NewAuthService(repo Repository, quota QuotaChecker) AuthService {
	return &authService{repo: repo, quota: quota}
}

func (s *authService) ValidateKey(ctx context.Context, rawKey string) (*models.User, error) {
	if err := validateKeyFormat(rawKey); err != nil {
		return nil, err
	}
	user, err := s.repo.FindUserByKeyHash(ctx, rawKey)
	if err != nil {
		return nil, ErrInvalidAPIKey
	}
	exceeded, err := s.quota.IsQuotaExceeded(ctx, user.ID, user.Plan)
	if err != nil {
		return nil, fmt.Errorf("auth.ValidateKey: %w", err)
	}
	if exceeded {
		return nil, ErrQuotaExceeded
	}
	return user, nil
}

func validateKeyFormat(key string) error {
	if key == "" {
		return ErrMissingAPIKey
	}
	if !strings.HasPrefix(key, keyPrefix) {
		return ErrInvalidKeyFormat
	}
	if len(key) != keyTotalLen {
		return ErrInvalidKeyLength
	}
	return nil
}
