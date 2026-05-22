// Package api provides shared infrastructure for the dashboard (/api/*) handlers.
package api

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thatsbass/veil/pkg/models"
)

// UserResolver maps a Clerk-authenticated email to a Veil user record.
// Dashboard handlers use this to obtain the Veil UUID from the Clerk session.
type UserResolver interface {
	ResolveUser(ctx context.Context, email string) (*models.User, error)
}

type pgUserResolver struct {
	db *pgxpool.Pool
}

func NewPGUserResolver(db *pgxpool.Pool) UserResolver {
	return &pgUserResolver{db: db}
}

func (r *pgUserResolver) ResolveUser(ctx context.Context, email string) (*models.User, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, email, plan, payment_customer_id, created_at, updated_at
		FROM users WHERE email = $1
	`, email)
	u := &models.User{}
	if err := row.Scan(&u.ID, &u.Email, &u.Plan, &u.PaymentCustomerID, &u.CreatedAt, &u.UpdatedAt); err != nil {
		return nil, fmt.Errorf("api.pgUserResolver.ResolveUser: %w", err)
	}
	return u, nil
}
