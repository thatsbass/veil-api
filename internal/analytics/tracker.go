package analytics

import (
	"context"
	"encoding/json"
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

type logPayload struct {
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"`
	Provider  string `json:"provider"`
	LatencyMS int    `json:"latency_ms"`
	Status    string `json:"status"`
}

func (t *redisTracker) publish(ctx context.Context, event Event) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if err := t.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: streamName,
		MaxLen: maxStreamLen,
		Approx: true,
		Values: map[string]any{
			"user_id":    event.UserID,
			"provider":   event.Provider,
			"format":     event.Format,
			"tokens_in":  fmt.Sprint(event.Usage.InputTokens),
			"tokens_out": fmt.Sprint(event.Usage.OutputTokens),
			"latency_ms": fmt.Sprint(event.LatencyMS),
			"status":     event.Status,
		},
	}).Err(); err != nil {
		return fmt.Errorf("analytics.redisTracker.publish xadd: %w", err)
	}

	payload, err := json.Marshal(logPayload{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Type:      event.Format,
		Provider:  event.Provider,
		LatencyMS: event.LatencyMS,
		Status:    event.Status,
	})
	if err != nil {
		return fmt.Errorf("analytics.redisTracker.publish marshal: %w", err)
	}
	if err := t.rdb.Publish(ctx, "logs:"+event.UserID, string(payload)).Err(); err != nil {
		return fmt.Errorf("analytics.redisTracker.publish pubsub: %w", err)
	}
	return nil
}
