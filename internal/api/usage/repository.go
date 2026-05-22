package usage

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// RequestRecord is a single historical request entry.
type RequestRecord struct {
	ID        string    `json:"id"          example:"550e8400-e29b-41d4-a716-446655440000"`
	Provider  string    `json:"provider"    example:"deepseek"`
	Format    string    `json:"format"      example:"anthropic"`
	TokensIn  int       `json:"tokens_in"   example:"120"`
	TokensOut int       `json:"tokens_out"  example:"340"`
	CostUSD   float64   `json:"cost_usd"    example:"0.000064"`
	LatencyMs *int      `json:"latency_ms"  example:"823"`
	Status    string    `json:"status"      example:"success"`
	CreatedAt time.Time `json:"created_at"  example:"2026-05-15T10:30:00Z"`
}

// UsageRepository reads historical request data from PostgreSQL.
type UsageRepository interface {
	GetHistory(ctx context.Context, userID string, limit int) ([]*RequestRecord, error)
}

type pgUsageRepository struct {
	db *pgxpool.Pool
}

func NewPGUsageRepository(db *pgxpool.Pool) UsageRepository {
	return &pgUsageRepository{db: db}
}

func (r *pgUsageRepository) GetHistory(ctx context.Context, userID string, limit int) ([]*RequestRecord, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, provider, format, tokens_in, tokens_out, cost_usd, latency_ms, status, created_at
		FROM requests
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("usage.pgUsageRepository.GetHistory: %w", err)
	}
	defer rows.Close()

	var records []*RequestRecord
	for rows.Next() {
		rec := &RequestRecord{}
		if err := rows.Scan(
			&rec.ID, &rec.Provider, &rec.Format,
			&rec.TokensIn, &rec.TokensOut, &rec.CostUSD,
			&rec.LatencyMs, &rec.Status, &rec.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("usage.pgUsageRepository.GetHistory scan: %w", err)
		}
		records = append(records, rec)
	}
	return records, nil
}
