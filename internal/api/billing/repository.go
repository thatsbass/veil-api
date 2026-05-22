package billing

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PlanRecord holds plan definition as stored in the plans table.
type PlanRecord struct {
	ID          string  `json:"id"           example:"free"`
	PriceUSD    float64 `json:"price_usd"    example:"0"`
	TokenQuota  int64   `json:"token_quota"  example:"100000"`
	MaxAPIKeys  int     `json:"max_api_keys" example:"1"`
}

// PlanRepository reads plan definitions and user plan assignments.
type PlanRepository interface {
	GetUserPlan(ctx context.Context, userID string) (*PlanRecord, error)
}

type pgPlanRepository struct {
	db *pgxpool.Pool
}

func NewPGPlanRepository(db *pgxpool.Pool) PlanRepository {
	return &pgPlanRepository{db: db}
}

func (r *pgPlanRepository) GetUserPlan(ctx context.Context, userID string) (*PlanRecord, error) {
	row := r.db.QueryRow(ctx, `
		SELECT p.id, p.price_usd, p.token_quota, p.max_api_keys
		FROM plans p
		JOIN users u ON u.plan = p.id
		WHERE u.id = $1
	`, userID)

	rec := &PlanRecord{}
	if err := row.Scan(&rec.ID, &rec.PriceUSD, &rec.TokenQuota, &rec.MaxAPIKeys); err != nil {
		return nil, fmt.Errorf("billing.pgPlanRepository.GetUserPlan: %w", err)
	}
	return rec, nil
}
