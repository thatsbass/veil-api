package keys

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thatsbass/veil/pkg/models"
)

var ErrKeyNotFound = errors.New("api key not found")

// KeyRepository manages API key CRUD scoped to a user.
type KeyRepository interface {
	Create(ctx context.Context, userID, name, keyHash string) (*models.APIKey, error)
	List(ctx context.Context, userID string) ([]*models.APIKey, error)
	Revoke(ctx context.Context, userID, keyID string) error
}

type pgKeyRepository struct {
	db *pgxpool.Pool
}

func NewPGKeyRepository(db *pgxpool.Pool) KeyRepository {
	return &pgKeyRepository{db: db}
}

func (r *pgKeyRepository) Create(ctx context.Context, userID, name, keyHash string) (*models.APIKey, error) {
	row := r.db.QueryRow(ctx, `
		INSERT INTO api_keys (user_id, key_hash, name)
		VALUES ($1, $2, $3)
		RETURNING id, user_id, key_hash, name, last_used, created_at
	`, userID, keyHash, name)

	k := &models.APIKey{}
	if err := row.Scan(&k.ID, &k.UserID, &k.KeyHash, &k.Name, &k.LastUsed, &k.CreatedAt); err != nil {
		return nil, fmt.Errorf("keys.pgKeyRepository.Create: %w", err)
	}
	return k, nil
}

func (r *pgKeyRepository) List(ctx context.Context, userID string) ([]*models.APIKey, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, key_hash, name, last_used, created_at
		FROM api_keys WHERE user_id = $1
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("keys.pgKeyRepository.List: %w", err)
	}
	defer rows.Close()

	var list []*models.APIKey
	for rows.Next() {
		k := &models.APIKey{}
		if err := rows.Scan(&k.ID, &k.UserID, &k.KeyHash, &k.Name, &k.LastUsed, &k.CreatedAt); err != nil {
			return nil, fmt.Errorf("keys.pgKeyRepository.List scan: %w", err)
		}
		list = append(list, k)
	}
	return list, nil
}

func (r *pgKeyRepository) Revoke(ctx context.Context, userID, keyID string) error {
	tag, err := r.db.Exec(ctx, `
		DELETE FROM api_keys WHERE id = $1 AND user_id = $2
	`, keyID, userID)
	if err != nil {
		return fmt.Errorf("keys.pgKeyRepository.Revoke: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrKeyNotFound
	}
	return nil
}

// scanKey is a helper used by Create when pgx returns no-rows on conflict.
func scanKey(row pgx.Row) (*models.APIKey, error) {
	k := &models.APIKey{}
	err := row.Scan(&k.ID, &k.UserID, &k.KeyHash, &k.Name, &k.LastUsed, &k.CreatedAt)
	return k, err
}
