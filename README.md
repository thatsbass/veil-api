# veil-api

[![Go](https://img.shields.io/badge/Go-1.24-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Swagger](https://img.shields.io/badge/API-Swagger-85ea2c)](docs/)

Veil API is the gateway at the core of the [Veil](https://veil.dev) platform. It
exposes the **Anthropic** and **OpenAI** API surfaces to local AI tools, then
translates and forwards requests to a configurable upstream model provider
(DeepSeek by default). It also provides the management API used by the
[`veil-cli`](../veil-cli) client and the
[`veil-dashboard`](../veil-dashboard) web app.

The gateway handles authentication, quota enforcement, token metering, and
billing, so that consumers only have to point their tools at a single endpoint.

## Table of contents

- [Features](#features)
- [How it works](#how-it-works)
- [Architecture](#architecture)
- [Prerequisites](#prerequisites)
- [Getting started](#getting-started)
- [API reference](#api-reference)
- [Plans and billing](#plans-and-billing)
- [Configuration](#configuration)
- [Development](#development)
- [Testing](#testing)
- [Contributing](#contributing)
- [License](#license)

## Features

- **Multi-format compatibility.** Implements the Anthropic Messages API, the
  OpenAI Chat Completions API, and the OpenAI Responses API on a single server.
- **Provider isolation.** The upstream client, payment processor, auth provider,
  and mailer are each implemented behind a small interface. Swapping any of them
  is a one-line change in `cmd/server/main.go`.
- **Token metering and quotas.** Monthly usage is tracked per user and enforced
  against plan limits; overage is billed at the metered rate.
- **Two authentication surfaces.** LLM endpoints use hashed API keys
  (`vl_live_xxx`); dashboard endpoints use Clerk JWTs resolved to user records.
- **Automatic migrations.** The SQL schema is versioned under `migrations/` and
  applied at startup.
- **Self-serve management.** API keys, usage, and billing endpoints back both
  the CLI and the web dashboard.
- **Generated documentation.** OpenAPI/Swagger specs are generated from code
  annotations.

## How it works

```
Claude CLI / Cursor / Aider / Codex
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

Request flow:

1. A local tool sends a request in its native format, authenticated with an API
   key (`Bearer vl_live_xxx`).
2. The gateway validates the key and resolves the associated user and plan.
3. The `translator` package converts the request to the upstream provider's
   format.
4. The `provider` client forwards the request to DeepSeek.
5. The response is translated back to the tool's expected format and returned.
6. Token usage is recorded and attributed to the user's plan.

## Architecture

```
internal/
  auth/              API key validation, quota enforcement
  auth/clerk/        Clerk JWT provider (swappable)
  billing/           Token metering, quota management
  billing/stripe/    Stripe webhook provider (swappable)
  api/keys/          Dashboard — API key management
  api/usage/         Dashboard — usage statistics
  api/billing/       Dashboard — plan and billing
  api/v1handler/     CLI-facing endpoints (/v1/usage, /v1/billing/plan)
  gateway/           Request routing and response translation
  mailer/            Transactional email interface
  mailer/resend/     Resend provider (swappable)
  provider/          Upstream LLM client (DeepSeek)
  translator/        Anthropic / OpenAI / Responses format adapters
  migrate/           Auto-migration at startup

cmd/server/          Server entry point (composition root)
cmd/seed/            Development seeder (test user + API key)
migrations/          Versioned SQL schema
pkg/models/          Shared domain models
docs/                Generated Swagger documentation
```

Providers are isolated in sub-packages behind interfaces. Swapping Stripe for
another payment processor, Clerk for another auth provider, or DeepSeek for
another model requires changing one line in `cmd/server/main.go`.

## Prerequisites

- Go 1.22 or later
- PostgreSQL 16
- Redis 7+
- A DeepSeek API key

## Getting started

```bash
# 1. Configure the environment
cp .env.example .env

# 2. Start PostgreSQL and Redis (optional if you run them locally)
make docker-up

# 3. Start the server — migrations run automatically
make run

# 4. Seed a test user and API key for local development
make seed
```

The API and Swagger UI are available at `http://localhost:3000`.

## API reference

### LLM endpoints — `Bearer vl_live_xxx`

| Method | Path | Format |
|--------|------|--------|
| POST | `/v1/messages` | Anthropic Messages API |
| POST | `/v1/chat/completions` | OpenAI Chat Completions |
| POST | `/v1/responses` | OpenAI Responses API |
| GET | `/v1/models` | Model list |
| GET | `/v1/usage` | Current month usage (CLI) |
| GET | `/v1/billing/plan` | Active plan (CLI) |

### Dashboard endpoints — `Bearer <Clerk JWT>`

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/keys` | Create an API key |
| GET | `/api/keys` | List API keys |
| DELETE | `/api/keys/:id` | Revoke an API key |
| GET | `/api/usage` | Current month usage |
| GET | `/api/usage/history` | Request history |
| GET | `/api/billing/plan` | Active plan |
| POST | `/api/billing/upgrade` | Upgrade plan |

### Public

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Liveness check |
| POST | `/webhooks/payment` | Payment provider webhooks |

## Plans and billing

| Plan | Price | Tokens / month | API keys |
|------|-------|----------------|----------|
| Free | $0 | 100k | 1 |
| Starter | $9 | 2M | 3 |
| Pro | $29 | 10M | 10 |
| Team | $99 | 50M | 50 |

Overage is billed at $0.50 per 1M tokens. Payments are processed through Stripe;
the `POST /webhooks/payment` endpoint keeps the metered usage and plan state in
sync with the payment provider.

## Configuration

Configuration is read from `.env` via `viper`, with environment variables taking
precedence.

| Variable | Description |
|----------|-------------|
| `PORT` | HTTP port (default: `3000`) |
| `HOST` | Swagger UI host (default: `localhost:3000`) |
| `BASE_URL` | Public base URL (default: `http://localhost:3000`) |
| `ENV` | `development` or `production` |
| `DATABASE_URL` | PostgreSQL connection string |
| `REDIS_URL` | Redis connection string |
| `DEEPSEEK_API_KEY` | DeepSeek secret key |
| `RESEND_API_KEY` | Resend email key (optional — noop if absent) |
| `STRIPE_SECRET_KEY` | Stripe secret key |
| `STRIPE_WEBHOOK_SECRET` | Stripe webhook signing secret |
| `CLERK_SECRET_KEY` | Clerk secret key (dashboard auth) |
| `API_KEY_SECRET` | Salt for API key hashing |

## Development

| Command | Description |
|---------|-------------|
| `make run` | Start the server |
| `make build` | Compile to `bin/veil` |
| `make test` | Run tests |
| `make test-race` | Run tests with the race detector |
| `make test-cover` | Generate an HTML coverage report |
| `make lint` | Run golangci-lint |
| `make migrate-up` / `migrate-down` | Apply / roll back migrations |
| `make migrate-create` | Scaffold a new numbered migration |
| `make sqlc` | Regenerate SQLC code from `sqlc/sqlc.yaml` |
| `make seed` | Insert a test user and API key |
| `make tidy` | Clean up `go.mod` |

### Docker

```bash
make docker-up                  # PostgreSQL + Redis
docker build -t veil .          # build the gateway image
docker run -p 3000:3000 --env-file .env veil
```

### Regenerating documentation

The Swagger documentation is generated from Go annotations in
`cmd/server/main.go`:

```bash
make swag-install   # one-time: install the swag CLI
make swag           # regenerate docs/
```

## Testing

```bash
make test
make test-race
make test-cover
```

## Contributing

- Follow the existing conventions: small interfaces, dependency injection, and
  isolated providers behind ports.
- Run `make lint` and `make test` before opening a pull request.
- Keep the Swagger annotations and this README in sync with code changes.

## License

MIT
