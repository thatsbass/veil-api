package billing

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

// Meter increments per-user token counters in Redis with a monthly TTL.
type Meter struct {
	rdb *redis.Client
}

func NewMeter(rdb *redis.Client) *Meter {
	return &Meter{rdb: rdb}
}

// IncrementAsync fires the Redis INCRBY in a goroutine so it never blocks the
// request path.
func (m *Meter) IncrementAsync(userID string, tokens int64) {
	go func() {
		if err := m.increment(context.Background(), userID, tokens); err != nil {
			log.Error().Err(err).Str("user_id", userID).Msg("meter increment failed")
		}
	}()
}

func (m *Meter) increment(ctx context.Context, userID string, tokens int64) error {
	key := usageKey(userID)
	pipe := m.rdb.Pipeline()
	pipe.IncrBy(ctx, key, tokens)
	// keep the key alive until the end of the current month
	pipe.ExpireAt(ctx, key, endOfMonth())
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("billing.Meter.increment: %w", err)
	}
	return nil
}

func usageKey(userID string) string {
	return fmt.Sprintf("usage:%s", userID)
}

func endOfMonth() time.Time {
	now := time.Now().UTC()
	// first day of next month at midnight
	return time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, time.UTC)
}
