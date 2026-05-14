package auth

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/thatsbass/veil/pkg/models"
	"github.com/thatsbass/veil/pkg/utils"
)

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
