CREATE TABLE plans (
    id           TEXT PRIMARY KEY,
    price_usd    DECIMAL(10, 2) NOT NULL,
    token_quota  BIGINT NOT NULL,
    max_api_keys INT NOT NULL
);

INSERT INTO plans (id, price_usd, token_quota, max_api_keys) VALUES
    ('free',    0,    100000,   1),
    ('starter', 9,   2000000,   3),
    ('pro',     29, 10000000,  10),
    ('team',    99, 50000000,  50);
