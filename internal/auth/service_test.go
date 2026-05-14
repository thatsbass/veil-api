package auth

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/thatsbass/veil/pkg/models"
)

// ---- mocks ----

type mockRepository struct{ mock.Mock }

func (m *mockRepository) FindUserByKeyHash(ctx context.Context, rawKey string) (*models.User, error) {
	args := m.Called(ctx, rawKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

type mockQuotaChecker struct{ mock.Mock }

func (m *mockQuotaChecker) IsQuotaExceeded(ctx context.Context, userID string, plan models.Plan) (bool, error) {
	args := m.Called(ctx, userID, plan)
	return args.Bool(0), args.Error(1)
}

// ---- helpers ----

// validKey builds a syntactically valid key (prefix + 40 chars = 48 total).
func validKey() string {
	return keyPrefix + strings.Repeat("a", keyTotalLen-len(keyPrefix))
}

func freeUser() *models.User {
	return &models.User{ID: "user-123", Email: "test@example.com", Plan: models.PlanFree}
}

// ---- tests ----

func TestValidateKey_Success(t *testing.T) {
	// Arrange
	repo := &mockRepository{}
	quota := &mockQuotaChecker{}
	svc := NewAuthService(repo, quota)

	repo.On("FindUserByKeyHash", mock.Anything, validKey()).Return(freeUser(), nil)
	quota.On("IsQuotaExceeded", mock.Anything, "user-123", models.PlanFree).Return(false, nil)

	// Act
	user, err := svc.ValidateKey(context.Background(), validKey())

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, "user-123", user.ID)
	repo.AssertExpectations(t)
	quota.AssertExpectations(t)
}

func TestValidateKey_EmptyKey_ReturnsMissingError(t *testing.T) {
	svc := NewAuthService(&mockRepository{}, &mockQuotaChecker{})

	user, err := svc.ValidateKey(context.Background(), "")

	assert.Nil(t, user)
	assert.ErrorIs(t, err, ErrMissingAPIKey)
}

func TestValidateKey_WrongPrefix_ReturnsFormatError(t *testing.T) {
	svc := NewAuthService(&mockRepository{}, &mockQuotaChecker{})

	user, err := svc.ValidateKey(context.Background(), "sk-"+strings.Repeat("a", 40))

	assert.Nil(t, user)
	assert.ErrorIs(t, err, ErrInvalidKeyFormat)
}

func TestValidateKey_WrongLength_ReturnsLengthError(t *testing.T) {
	svc := NewAuthService(&mockRepository{}, &mockQuotaChecker{})

	user, err := svc.ValidateKey(context.Background(), keyPrefix+"short")

	assert.Nil(t, user)
	assert.ErrorIs(t, err, ErrInvalidKeyLength)
}

func TestValidateKey_KeyNotInDB_ReturnsInvalidKeyError(t *testing.T) {
	// Arrange
	repo := &mockRepository{}
	svc := NewAuthService(repo, &mockQuotaChecker{})

	repo.On("FindUserByKeyHash", mock.Anything, validKey()).Return(nil, errors.New("not found"))

	// Act
	user, err := svc.ValidateKey(context.Background(), validKey())

	// Assert
	assert.Nil(t, user)
	assert.ErrorIs(t, err, ErrInvalidAPIKey)
	repo.AssertExpectations(t)
}

func TestValidateKey_QuotaExceeded_ReturnsQuotaError(t *testing.T) {
	// Arrange
	repo := &mockRepository{}
	quota := &mockQuotaChecker{}
	svc := NewAuthService(repo, quota)

	repo.On("FindUserByKeyHash", mock.Anything, validKey()).Return(freeUser(), nil)
	quota.On("IsQuotaExceeded", mock.Anything, "user-123", models.PlanFree).Return(true, nil)

	// Act
	user, err := svc.ValidateKey(context.Background(), validKey())

	// Assert
	assert.Nil(t, user)
	assert.ErrorIs(t, err, ErrQuotaExceeded)
	repo.AssertExpectations(t)
	quota.AssertExpectations(t)
}

func TestValidateKey_QuotaCheckFails_WrapsError(t *testing.T) {
	// Arrange
	repo := &mockRepository{}
	quota := &mockQuotaChecker{}
	svc := NewAuthService(repo, quota)

	redisErr := errors.New("redis: connection refused")
	repo.On("FindUserByKeyHash", mock.Anything, validKey()).Return(freeUser(), nil)
	quota.On("IsQuotaExceeded", mock.Anything, "user-123", models.PlanFree).Return(false, redisErr)

	// Act
	user, err := svc.ValidateKey(context.Background(), validKey())

	// Assert
	assert.Nil(t, user)
	assert.ErrorContains(t, err, "auth.ValidateKey")
	assert.ErrorIs(t, err, redisErr)
	repo.AssertExpectations(t)
	quota.AssertExpectations(t)
}
