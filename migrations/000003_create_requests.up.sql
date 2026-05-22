CREATE TABLE IF NOT EXISTS requests (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id),
    provider    TEXT NOT NULL,
    format      TEXT NOT NULL,
    tokens_in   INT NOT NULL DEFAULT 0,
    tokens_out  INT NOT NULL DEFAULT 0,
    cost_usd    DECIMAL(10, 6) NOT NULL DEFAULT 0,
    latency_ms  INT,
    status      TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_requests_user_date ON requests(user_id, created_at DESC);
