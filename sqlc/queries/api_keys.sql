-- name: CreateAPIKey :one
INSERT INTO api_keys (user_id, key_hash, name)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetAPIKeyByHash :one
SELECT k.*, u.plan, u.email
FROM api_keys k
JOIN users u ON u.id = k.user_id
WHERE k.key_hash = $1;

-- name: ListAPIKeysByUser :many
SELECT * FROM api_keys WHERE user_id = $1 ORDER BY created_at DESC;

-- name: DeleteAPIKey :exec
DELETE FROM api_keys WHERE id = $1 AND user_id = $2;

-- name: TouchAPIKey :exec
UPDATE api_keys SET last_used = NOW() WHERE key_hash = $1;
