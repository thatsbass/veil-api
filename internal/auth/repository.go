package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/thatsbass/veil/pkg/models"
	"github.com/thatsbass/veil/pkg/utils"
)

const apiKeyCacheTTL = 5 * time.Minute

// Repository looks up users by their hashed API key.
type Repository interface {
	FindUserByKeyHash(ctx context.Context, rawKey string) (*models.User, error)
}

// QuotaChecker reports whether a user has exhausted their monthly token quota.
type QuotaChecker interface {
	IsQuotaExceeded(ctx context.Context, userID string, plan models.Plan) (bool, error)
}

// --- PostgreSQL repository ---

type pgRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) Repository {
	return &pgRepository{db: db}
}

func (r *pgRepository) FindUserByKeyHash(ctx context.Context, rawKey string) (*models.User, error) {
	hash := utils.HashAPIKey(rawKey)
	row := r.db.QueryRow(ctx, `
		SELECT u.id, u.email, u.plan, u.stripe_id, u.created_at, u.updated_at
		FROM api_keys k
		JOIN users u ON u.id = k.user_id
		WHERE k.key_hash = $1
	`, hash)

	user := &models.User{}
	if err := row.Scan(&user.ID, &user.Email, &user.Plan, &user.StripeID, &user.CreatedAt, &user.UpdatedAt); err != nil {
		return nil, fmt.Errorf("auth.pgRepository.FindUserByKeyHash: %w", err)
	}

	go r.touchKey(context.Background(), hash)
	return user, nil
}

func (r *pgRepository) touchKey(ctx context.Context, hash string) {
	_, _ = r.db.Exec(ctx, `UPDATE api_keys SET last_used = NOW() WHERE key_hash = $1`, hash)
}

// --- Cached repository (Redis L1 + PostgreSQL L2) ---

type cachedRepository struct {
	pg  *pgRepository
	rdb *redis.Client
}

// NewCachedRepository returns a Repository that checks Redis before hitting PostgreSQL.
func NewCachedRepository(db *pgxpool.Pool, rdb *redis.Client) Repository {
	return &cachedRepository{
		pg:  &pgRepository{db: db},
		rdb: rdb,
	}
}

func (r *cachedRepository) FindUserByKeyHash(ctx context.Context, rawKey string) (*models.User, error) {
	hash := utils.HashAPIKey(rawKey)
	cacheKey := apiKeyCacheKey(hash)

	if user, err := r.fromCache(ctx, cacheKey); err == nil {
		return user, nil
	}

	user, err := r.pg.FindUserByKeyHash(ctx, rawKey)
	if err != nil {
		return nil, err
	}

	r.setCache(ctx, cacheKey, user)
	return user, nil
}

func (r *cachedRepository) fromCache(ctx context.Context, key string) (*models.User, error) {
	raw, err := r.rdb.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}
	var user models.User
	if err := json.Unmarshal(raw, &user); err != nil {
		return nil, fmt.Errorf("auth.cachedRepository.fromCache: %w", err)
	}
	return &user, nil
}

func (r *cachedRepository) setCache(ctx context.Context, key string, user *models.User) {
	raw, err := json.Marshal(user)
	if err != nil {
		return
	}
	_ = r.rdb.Set(ctx, key, raw, apiKeyCacheTTL).Err()
}

func apiKeyCacheKey(hash string) string {
	return fmt.Sprintf("auth:key:%s", hash)
}

// --- Redis quota checker ---

type redisQuotaChecker struct {
	rdb *redis.Client
}

func NewRedisQuotaChecker(rdb *redis.Client) QuotaChecker {
	return &redisQuotaChecker{rdb: rdb}
}

func (q *redisQuotaChecker) IsQuotaExceeded(ctx context.Context, userID string, plan models.Plan) (bool, error) {
	key := fmt.Sprintf("usage:%s", userID)
	val, err := q.rdb.Get(ctx, key).Int64()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("auth.redisQuotaChecker.IsQuotaExceeded: %w", err)
	}
	return val >= models.PlanQuota(plan), nil
}
