package analytics

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

const (
	streamName    = "veil:analytics"
	maxStreamLen  = 100_000
)

type redisTracker struct {
	rdb *redis.Client
}

// NewRedisTracker publishes events to a Redis Stream.
// Consumers can process them asynchronously for analytics and billing sync.
func NewRedisTracker(rdb *redis.Client) AnalyticsService {
	return &redisTracker{rdb: rdb}
}

func (t *redisTracker) Track(event Event) {
	go func() {
		if err := t.publish(context.Background(), event); err != nil {
			log.Error().Err(err).Msg("analytics: failed to publish event")
		}
	}()
}

func (t *redisTracker) publish(ctx context.Context, event Event) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	err := t.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: streamName,
		MaxLen: maxStreamLen,
		Approx: true,
		Values: map[string]any{
			"user_id":      event.UserID,
			"provider":     event.Provider,
			"format":       event.Format,
			"tokens_in":    fmt.Sprint(event.Usage.InputTokens),
			"tokens_out":   fmt.Sprint(event.Usage.OutputTokens),
		},
	}).Err()
	if err != nil {
		return fmt.Errorf("analytics.redisTracker.publish: %w", err)
	}
	return nil
}
