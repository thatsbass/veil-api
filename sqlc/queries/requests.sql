-- name: CreateRequest :one
INSERT INTO requests (user_id, provider, format, tokens_in, tokens_out, cost_usd, latency_ms, status)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetMonthlyUsage :one
SELECT
    COALESCE(SUM(tokens_in + tokens_out), 0)::BIGINT AS total_tokens,
    COALESCE(SUM(cost_usd), 0)                       AS total_cost_usd
FROM requests
WHERE user_id = $1
  AND created_at >= date_trunc('month', NOW());

-- name: ListRequestsByUser :many
SELECT * FROM requests
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;
