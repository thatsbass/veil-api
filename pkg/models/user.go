package models

import "time"

type Plan string

const (
	PlanFree    Plan = "free"
	PlanStarter Plan = "starter"
	PlanPro     Plan = "pro"
	PlanTeam    Plan = "team"
)

type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Plan      Plan      `json:"plan"`
	StripeID  *string   `json:"stripe_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type APIKey struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	KeyHash   string     `json:"-"`
	Name      string     `json:"name"`
	LastUsed  *time.Time `json:"last_used,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// PlanQuota returns the monthly token quota for a plan.
func PlanQuota(p Plan) int64 {
	switch p {
	case PlanStarter:
		return 2_000_000
	case PlanPro:
		return 10_000_000
	case PlanTeam:
		return 50_000_000
	default:
		return 100_000
	}
}
