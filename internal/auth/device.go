package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const deviceSessionTTL = 10 * time.Minute

// DeviceSession holds the state of a pending CLI authentication.
type DeviceSession struct {
	DeviceCode string    `json:"device_code"`
	UserCode   string    `json:"user_code"`
	ExpiresAt  time.Time `json:"expires_at"`
	Confirmed  bool      `json:"confirmed"`
	UserID     string    `json:"user_id,omitempty"`
	APIKey     string    `json:"api_key,omitempty"` // raw key, present only after confirmation
}

// DeviceStore manages device auth sessions.
type DeviceStore interface {
	Create(ctx context.Context) (*DeviceSession, error)
	GetByDeviceCode(ctx context.Context, deviceCode string) (*DeviceSession, error)
	GetByUserCode(ctx context.Context, userCode string) (*DeviceSession, error)
	Confirm(ctx context.Context, userCode, userID, rawAPIKey string) error
}

type redisDeviceStore struct {
	rdb *redis.Client
}

// NewRedisDeviceStore creates a DeviceStore backed by Redis.
func NewRedisDeviceStore(rdb *redis.Client) DeviceStore {
	return &redisDeviceStore{rdb: rdb}
}

func (s *redisDeviceStore) Create(ctx context.Context) (*DeviceSession, error) {
	deviceCode, err := generateDeviceCode()
	if err != nil {
		return nil, fmt.Errorf("device.Create: %w", err)
	}
	userCode, err := generateUserCode()
	if err != nil {
		return nil, fmt.Errorf("device.Create: %w", err)
	}

	sess := &DeviceSession{
		DeviceCode: deviceCode,
		UserCode:   userCode,
		ExpiresAt:  time.Now().Add(deviceSessionTTL),
	}

	data, err := json.Marshal(sess)
	if err != nil {
		return nil, fmt.Errorf("device.Create marshal: %w", err)
	}

	pipe := s.rdb.Pipeline()
	pipe.Set(ctx, deviceCodeKey(deviceCode), data, deviceSessionTTL)
	pipe.Set(ctx, userCodeKey(userCode), deviceCode, deviceSessionTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		return nil, fmt.Errorf("device.Create redis: %w", err)
	}
	return sess, nil
}

func (s *redisDeviceStore) GetByDeviceCode(ctx context.Context, deviceCode string) (*DeviceSession, error) {
	data, err := s.rdb.Get(ctx, deviceCodeKey(deviceCode)).Bytes()
	if err == redis.Nil {
		return nil, ErrDeviceSessionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("device.GetByDeviceCode: %w", err)
	}
	var sess DeviceSession
	if err := json.Unmarshal(data, &sess); err != nil {
		return nil, fmt.Errorf("device.GetByDeviceCode unmarshal: %w", err)
	}
	return &sess, nil
}

func (s *redisDeviceStore) GetByUserCode(ctx context.Context, userCode string) (*DeviceSession, error) {
	deviceCode, err := s.rdb.Get(ctx, userCodeKey(userCode)).Result()
	if err == redis.Nil {
		return nil, ErrDeviceSessionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("device.GetByUserCode: %w", err)
	}
	return s.GetByDeviceCode(ctx, deviceCode)
}

func (s *redisDeviceStore) Confirm(ctx context.Context, userCode, userID, rawAPIKey string) error {
	sess, err := s.GetByUserCode(ctx, userCode)
	if err != nil {
		return err
	}
	if time.Now().After(sess.ExpiresAt) {
		return ErrDeviceSessionExpired
	}

	sess.Confirmed = true
	sess.UserID = userID
	sess.APIKey = rawAPIKey

	data, err := json.Marshal(sess)
	if err != nil {
		return fmt.Errorf("device.Confirm marshal: %w", err)
	}

	ttl := time.Until(sess.ExpiresAt)
	return s.rdb.Set(ctx, deviceCodeKey(sess.DeviceCode), data, ttl).Err()
}

// --- errors ---

var (
	ErrDeviceSessionNotFound = fmt.Errorf("device session not found or expired")
	ErrDeviceSessionExpired  = fmt.Errorf("device session expired")
)

// --- helpers ---

func deviceCodeKey(code string) string { return "device:code:" + code }
func userCodeKey(code string) string   { return "device:user:" + code }

func generateDeviceCode() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
