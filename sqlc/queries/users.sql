-- name: CreateUser :one
INSERT INTO users (email, plan)
VALUES ($1, $2)
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: UpdateUserPlan :one
UPDATE users
SET plan = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateUserStripeID :one
UPDATE users
SET stripe_id = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;
