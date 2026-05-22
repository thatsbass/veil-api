# Veil

Use your existing tools — Claude CLI, Cursor, Aider, Codex — with cheaper models in the background. Same endpoints. Same workflow. Up to 90% less in API costs.

---

## How it works

Veil sits between your tools and the model provider. It exposes the Anthropic and OpenAI API surfaces, intercepts requests, translates them to DeepSeek (or another provider), and returns responses in the format your tool expects. The tool never knows.

```
Claude CLI / Cursor / Aider
         |
  Bearer vl_live_xxx
         |
      Veil API
    /v1/messages
    /v1/chat/completions
    /v1/responses
         |
    DeepSeek API
```

---

## Requirements

- Go 1.22+
- PostgreSQL 16
- Redis 7+
- A DeepSeek API key

---

## Getting started

Copy the environment file and fill in your values.

```bash
cp .env.example .env
```

Start the server. Migrations run automatically at startup.

```bash
make run
```

Seed a test API key for local development.

```bash
make seed
```

The API and Swagger UI are available at `http://localhost:3000`.

---

## API

### LLM endpoints — `Bearer vl_live_xxx`

| Method | Path | Format |
|--------|------|--------|
| POST | `/v1/messages` | Anthropic Messages API |
| POST | `/v1/chat/completions` | OpenAI Chat Completions |
| POST | `/v1/responses` | OpenAI Responses API |
| GET | `/v1/models` | Model list |

### Dashboard endpoints — `Bearer <Clerk JWT>`

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/keys` | Create API key |
| GET | `/api/keys` | List API keys |
| DELETE | `/api/keys/:id` | Revoke API key |
| GET | `/api/usage` | Current month usage |
| GET | `/api/usage/history` | Request history |
| GET | `/api/billing/plan` | Active plan |
| POST | `/api/billing/upgrade` | Upgrade plan |

### Public

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Liveness check |
| POST | `/webhooks/payment` | Payment provider webhooks |

---

## Plans

| Plan | Price | Tokens / month | API keys |
|------|-------|----------------|----------|
| Free | $0 | 100k | 1 |
| Starter | $9 | 2M | 3 |
| Pro | $29 | 10M | 10 |
| Team | $99 | 50M | 50 |

Overage is billed at $0.50 per 1M tokens.

---

## Development

```bash
make run          # start the server
make test         # run tests
make test-race    # run tests with race detector
make seed         # insert a test user and API key
make migrate-up   # apply pending migrations
make migrate-down # roll back one migration
make swag         # regenerate Swagger docs
make build        # compile to bin/veil
make lint         # run golangci-lint
```

---

## Environment variables

| Variable | Description |
|----------|-------------|
| `PORT` | HTTP port (default: 8080) |
| `HOST` | Swagger UI host (default: localhost:8080) |
| `ENV` | `development` or `production` |
| `DATABASE_URL` | PostgreSQL connection string |
| `REDIS_URL` | Redis connection string |
| `DEEPSEEK_API_KEY` | DeepSeek secret key |
| `RESEND_API_KEY` | Resend email key (optional — noop if absent) |
| `STRIPE_SECRET_KEY` | Stripe secret key |
| `STRIPE_WEBHOOK_SECRET` | Stripe webhook signing secret |
| `CLERK_SECRET_KEY` | Clerk secret key (dashboard auth) |
| `API_KEY_SECRET` | Salt for API key hashing |

---

## Docker

```bash
docker build -t veil .
docker run -p 8080:8080 --env-file .env veil
```

---

## Architecture

```
internal/
  auth/           API key validation, quota enforcement
  auth/clerk/     Clerk JWT provider (swappable)
  billing/        Token metering, quota management
  billing/stripe/ Stripe webhook provider (swappable)
  api/keys/       Dashboard — API key management
  api/usage/      Dashboard — usage statistics
  api/billing/    Dashboard — plan and billing
  gateway/        Request routing and response translation
  mailer/         Transactional email interface
  mailer/resend/  Resend provider (swappable)
  provider/       Upstream LLM client (DeepSeek)
  translator/     Anthropic / OpenAI / Responses format adapters
  migrate/        Auto-migration at startup
```

Providers are isolated in sub-packages behind interfaces. Swapping Stripe for another payment processor, Clerk for another auth provider, or DeepSeek for another model requires changing one line in `main.go`.

---

## License

MIT
