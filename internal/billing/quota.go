package billing

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// QuotaManager reads current usage from Redis.
type QuotaManager struct {
	rdb *redis.Client
}

func NewQuotaManager(rdb *redis.Client) *QuotaManager {
	return &QuotaManager{rdb: rdb}
}

func (q *QuotaManager) GetUsage(ctx context.Context, userID string) (int64, error) {
	val, err := q.rdb.Get(ctx, usageKey(userID)).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("billing.QuotaManager.GetUsage: %w", err)
	}
	return val, nil
}
