# Veil — CLAUDE.md

> Ce fichier est la source de vérité pour Claude Code.
> Lis-le entièrement avant toute modification du projet.

---

## Vision du produit

**Veil** est un gateway universel qui permet aux développeurs d'utiliser leurs
outils habituels (Claude CLI, Codex, Cursor, Aider) avec des modèles moins chers
en arrière-plan (DeepSeek, Groq, Mistral), de manière totalement transparente.

**Proposition de valeur :**
> "Garde ton workflow. Réduis ta facture API de 90%."

**Ce que fait Veil :**
- Expose les mêmes endpoints qu'Anthropic et OpenAI
- Intercepte les requêtes des outils (Claude CLI, Codex...)
- Traduit et route vers DeepSeek (ou autre provider)
- Retourne la réponse dans le format attendu par l'outil
- L'utilisateur final ne sait pas quel modèle est utilisé en coulisse

---

## Architecture globale

```
Claude CLI / Codex / Cursor / Aider
              │
         Clé Veil : vl_live_xxx
              │
    ┌─────────▼──────────┐
    │    VEIL SERVER     │  ← Go + Fiber (ce repo)
    │   api.veil.dev     │
    ├────────────────────┤
    │ Auth   │ Translate │
    │ Meter  │ Provider  │
    │ Billing│ Analytics │
    └─────────┬──────────┘
              │  (clés secrètes côté serveur)
         DeepSeek API
```

---

## Stack technique

| Composant       | Choix              | Version  |
|-----------------|--------------------|----------|
| Language        | Go                 | 1.22+    |
| HTTP Framework  | Fiber              | v2       |
| SQL             | sqlc + pgx         | latest   |
| Migrations      | golang-migrate     | v4       |
| Cache           | go-redis           | v9       |
| Config          | viper              | v1       |
| Logs            | zerolog            | v1       |
| HTTP Client     | resty              | v2       |
| Tests           | testify            | v1       |
| Container       | Docker + Dokploy   | -        |
| DB              | PostgreSQL 16      | Supabase |
| Cache infra     | Redis              | Upstash  |
| Dashboard       | Next.js 14         | Vercel   |
| Paiements       | Stripe             | -        |
| CDN             | Cloudflare         | -        |

---

## Structure du projet

```
veil/
├── cmd/
│   └── server/
│       └── main.go                  ← Point d'entrée
│
├── internal/                        ← Code privé, jamais importé dehors
│   ├── auth/
│   │   ├── service.go               ← Interface AuthService
│   │   ├── middleware.go            ← Fiber middleware
│   │   └── repository.go           ← Redis + PostgreSQL
│   │
│   ├── gateway/
│   │   ├── handler.go               ← HTTP handlers
│   │   ├── streaming.go             ← SSE / streaming
│   │   └── detector.go             ← Détection format entrant
│   │
│   ├── translator/
│   │   ├── interface.go             ← Interface Translator
│   │   ├── anthropic.go             ← Anthropic → DeepSeek
│   │   ├── openai.go                ← OpenAI → DeepSeek
│   │   └── responses.go            ← Responses API → DeepSeek
│   │
│   ├── provider/
│   │   ├── interface.go             ← Interface Provider
│   │   ├── deepseek.go              ← Client DeepSeek (MVP)
│   │   └── health.go               ← Health checks
│   │
│   ├── billing/
│   │   ├── service.go               ← Interface BillingService
│   │   ├── meter.go                 ← Comptage tokens
│   │   ├── quota.go                 ← Gestion quotas
│   │   └── stripe.go               ← Webhooks Stripe
│   │
│   ├── analytics/
│   │   ├── service.go               ← Interface AnalyticsService
│   │   └── tracker.go              ← Events async
│   │
│   └── config/
│       └── config.go               ← Config centralisée (viper)
│
├── pkg/                             ← Code partageable
│   ├── models/
│   │   ├── request.go              ← Types de requêtes
│   │   ├── response.go             ← Types de réponses
│   │   └── user.go                 ← Types utilisateur
│   └── utils/
│       ├── hash.go                 ← SHA256 helpers
│       └── tokens.go              ← Comptage tokens
│
├── migrations/                      ← Fichiers SQL golang-migrate
│   ├── 000001_create_users.up.sql
│   ├── 000001_create_users.down.sql
│   ├── 000002_create_api_keys.up.sql
│   ├── 000002_create_api_keys.down.sql
│   ├── 000003_create_requests.up.sql
│   ├── 000003_create_requests.down.sql
│   └── 000004_create_billing.up.sql
│
├── sqlc/                            ← Queries SQL pour sqlc
│   ├── sqlc.yaml
│   └── queries/
│       ├── users.sql
│       ├── api_keys.sql
│       └── requests.sql
│
├── .claude/                         ← Config Claude Code
│   ├── commands/
│   │   ├── new-provider.md
│   │   ├── new-migration.md
│   │   └── test-coverage.md
│   └── settings.json
│
├── .env.example                     ← Variables d'env (jamais .env en git)
├── docker-compose.yml               ← Dev local complet
├── Dockerfile                       ← Build production
├── fly.toml                         ← Config Fly.io (backup)
├── go.mod
├── go.sum
└── CLAUDE.md                        ← CE FICHIER
```

---

## Conventions de code

### Nommage

```go
// Interfaces → suffixe "Service" ou "Repository"
type AuthService interface {}
type UserRepository interface {}

// Implémentations → préfixe du contexte
type redisAuthService struct {}
type pgUserRepository struct {}

// Handlers → suffixe "Handler"
func (h *Handler) HandleMessages(c *fiber.Ctx) error {}

// Erreurs → préfixe "Err"
var ErrInvalidAPIKey = errors.New("invalid api key")
var ErrQuotaExceeded = errors.New("quota exceeded")
```

### Gestion des erreurs

```go
// TOUJOURS wrapper les erreurs avec du contexte
if err != nil {
    return fmt.Errorf("auth.ValidateKey: %w", err)
}

// JAMAIS ignorer une erreur
result, _ := someFunc() // ← INTERDIT

// Logger avec zerolog, pas fmt.Println
log.Error().Err(err).Str("user_id", userID).Msg("failed to validate key")
```

### Contexte

```go
// TOUJOURS passer le contexte en premier paramètre
func (s *service) DoSomething(ctx context.Context, id string) error {}

// Respecter les timeouts
ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
defer cancel()
```

### Injection de dépendances

```go
// TOUJOURS injecter via les interfaces, jamais les structs concrètes
type Handler struct {
    auth     auth.AuthService       // interface ✓
    billing  billing.BillingService // interface ✓
    provider provider.Provider      // interface ✓
}

// Jamais :
type Handler struct {
    auth *redisAuthService // struct concrète ✗
}
```

---

## Clean Code — Principes fondamentaux

> **Règle d'or** : Le code est lu 10x plus souvent qu'il est écrit.
> Écris pour le prochain développeur, pas pour la machine.

---

### 1. Une fonction = une seule responsabilité

```go
// ✗ INTERDIT — fonction qui fait trop de choses
func (s *service) HandleRequest(c *fiber.Ctx) error {
    // valide la clé API
    key := c.Get("Authorization")
    hash := sha256.Sum256([]byte(key))
    user, err := s.db.QueryRow("SELECT * FROM users WHERE key_hash = $1", hash)
    // vérifie le quota
    usage, _ := s.redis.Get("usage:" + user.ID)
    if usage > user.Plan.Quota {
        return c.Status(429).JSON(...)
    }
    // traduit la requête
    var body AnthropicRequest
    c.BodyParser(&body)
    dsReq := DeepSeekRequest{Messages: convertMessages(body.Messages)}
    // appelle DeepSeek
    resp, _ := http.Post("https://api.deepseek.com/...", dsReq)
    // compte les tokens
    s.redis.IncrBy("tokens:"+user.ID, resp.Usage.Total)
    // retourne la réponse
    return c.JSON(convertResponse(resp))
}

// ✓ CORRECT — chaque fonction fait une seule chose
func (h *Handler) HandleMessages(c *fiber.Ctx) error {
    user := c.Locals("user").(*models.User)       // auth déjà fait par middleware
    req, err := h.translator.ParseRequest(c)       // parsing
    if err != nil {
        return h.respondError(c, err)
    }
    resp, err := h.provider.Complete(c.Context(), req)  // appel provider
    if err != nil {
        return h.respondError(c, err)
    }
    go h.billing.RecordUsage(user.ID, resp.Usage)  // billing async
    return h.translator.WriteResponse(c, resp)     // réponse
}
```

---

### 2. Pas de nombres ou strings magiques

```go
// ✗ INTERDIT — qu'est-ce que 429 ? qu'est-ce que "free" ?
if usage > 100000 {
    return c.Status(429).JSON(fiber.Map{"error": "quota exceeded"})
}

// ✓ CORRECT — tout est nommé et explicite
const (
    FreePlanTokenQuota = 100_000
    StatusQuotaExceeded = fiber.StatusTooManyRequests
)

var ErrQuotaExceeded = errors.New("monthly token quota exceeded")

if usage > FreePlanTokenQuota {
    return c.Status(StatusQuotaExceeded).JSON(newErrorResponse(ErrQuotaExceeded))
}
```

---

### 3. Noms explicites — pas d'abréviations obscures

```go
// ✗ INTERDIT
func chkQ(uid string, tkns int) bool {}
func proc(r *http.Request) {}
var u *User
var tmp interface{}
var d []byte

// ✓ CORRECT
func hasRemainingQuota(userID string, tokensRequested int) bool {}
func processCompletionRequest(r *http.Request) {}
var currentUser *User
var responseBody []byte
```

---

### 4. Fonctions courtes — max 30 lignes

```go
// ✗ INTERDIT — fonction de 80 lignes avec 5 niveaux d'imbrication
func (s *service) ValidateAndProcess(...) error {
    if key != "" {
        if len(key) > 10 {
            if strings.HasPrefix(key, "vl_live_") {
                hash := sha256Hash(key)
                user, err := s.repo.FindByHash(hash)
                if err != nil {
                    if errors.Is(err, pgx.ErrNoRows) {
                        // ... 50 autres lignes
                    }
                }
            }
        }
    }
}

// ✓ CORRECT — extraire en fonctions bien nommées
func (s *service) ValidateKey(ctx context.Context, key string) (*User, error) {
    if err := validateKeyFormat(key); err != nil {
        return nil, err
    }
    return s.repo.FindByKeyHash(ctx, sha256Hash(key))
}

func validateKeyFormat(key string) error {
    if key == "" {
        return ErrMissingAPIKey
    }
    if !strings.HasPrefix(key, "vl_live_") {
        return ErrInvalidKeyFormat
    }
    if len(key) != 48 {
        return ErrInvalidKeyLength
    }
    return nil
}
```

---

### 5. Early return — pas de else après un return

```go
// ✗ INTERDIT — pyramide de if/else
func processRequest(req *Request) (*Response, error) {
    if req != nil {
        if req.UserID != "" {
            if req.Tokens > 0 {
                // logique principale enfouie profondément
                return doWork(req)
            } else {
                return nil, ErrNoTokens
            }
        } else {
            return nil, ErrNoUserID
        }
    } else {
        return nil, ErrNilRequest
    }
}

// ✓ CORRECT — early return, logique principale en évidence
func processRequest(req *Request) (*Response, error) {
    if req == nil {
        return nil, ErrNilRequest
    }
    if req.UserID == "" {
        return nil, ErrNoUserID
    }
    if req.Tokens <= 0 {
        return nil, ErrNoTokens
    }
    return doWork(req) // logique principale claire et visible
}
```

---

### 6. Pas de commentaires inutiles — le code se documente lui-même

```go
// ✗ INTERDIT — commentaire qui répète le code
// Incrémente le compteur de tokens de l'utilisateur
s.redis.IncrBy("tokens:"+userID, tokenCount)

// ✗ INTERDIT — code mort commenté
// func oldValidation(key string) bool {
//     return len(key) > 5
// }

// ✓ CORRECT — commentaire qui explique le POURQUOI, pas le QUOI
// DeepSeek retourne parfois les tool_calls en XML au lieu de JSON.
// On normalise ici avant de continuer le traitement.
normalized := normalizeToolCalls(rawResponse)

// ✓ CORRECT — godoc sur les fonctions exportées
// ValidateKey vérifie qu'une clé API existe et que son quota
// n'est pas épuisé. Retourne ErrQuotaExceeded si la limite
// mensuelle est atteinte.
func (s *authService) ValidateKey(ctx context.Context, key string) (*User, error) {}
```

---

### 7. Pas de code dupliqué — DRY (Don't Repeat Yourself)

```go
// ✗ INTERDIT — même logique répétée dans 3 handlers
func (h *Handler) HandleMessages(c *fiber.Ctx) error {
    if err != nil {
        h.log.Error().Err(err).Msg("handler error")
        return c.Status(500).JSON(fiber.Map{"error": err.Error()})
    }
}
func (h *Handler) HandleCompletions(c *fiber.Ctx) error {
    if err != nil {
        h.log.Error().Err(err).Msg("handler error")
        return c.Status(500).JSON(fiber.Map{"error": err.Error()})
    }
}

// ✓ CORRECT — une seule fonction réutilisable
func (h *Handler) respondError(c *fiber.Ctx, err error) error {
    status, body := mapErrorToResponse(err)
    h.log.Error().Err(err).Int("status", status).Msg("request failed")
    return c.Status(status).JSON(body)
}

func mapErrorToResponse(err error) (int, fiber.Map) {
    switch {
    case errors.Is(err, ErrInvalidAPIKey):
        return fiber.StatusUnauthorized, fiber.Map{"error": "invalid api key"}
    case errors.Is(err, ErrQuotaExceeded):
        return fiber.StatusTooManyRequests, fiber.Map{"error": "quota exceeded"}
    default:
        return fiber.StatusInternalServerError, fiber.Map{"error": "internal error"}
    }
}
```

---

### 8. Structs lisibles — pas plus de 5 champs sans groupement

```go
// ✗ INTERDIT — struct avec 12 champs plats
type Request struct {
    UserID      string
    APIKey      string
    Model       string
    Messages    []Message
    MaxTokens   int
    Temperature float64
    TopP        float64
    Stream      bool
    Provider    string
    Format      string
    RequestID   string
    Timestamp   time.Time
}

// ✓ CORRECT — groupé logiquement
type CompletionRequest struct {
    Meta     RequestMeta
    Config   ModelConfig
    Payload  RequestPayload
}

type RequestMeta struct {
    UserID    string
    RequestID string
    Format    string    // anthropic | openai | responses
    Timestamp time.Time
}

type ModelConfig struct {
    Model       string
    MaxTokens   int
    Temperature float64
    TopP        float64
    Stream      bool
}

type RequestPayload struct {
    Messages []Message
    System   string
    Tools    []Tool
}
```

---

### 9. Tests lisibles — AAA pattern (Arrange, Act, Assert)

```go
// ✓ CORRECT — structure claire et répétable
func TestAuthService_ValidateKey_QuotaExceeded(t *testing.T) {
    // Arrange — préparer le contexte
    mockRepo := &MockUserRepository{}
    mockRedis := &MockRedisClient{}
    service := NewAuthService(mockRepo, mockRedis)

    user := &models.User{ID: "user-123", Plan: "free"}
    mockRepo.On("FindByKeyHash", mock.Anything, mock.Anything).Return(user, nil)
    mockRedis.On("Get", "usage:user-123").Return("150000", nil) // > 100k free limit

    // Act — exécuter l'action testée
    result, err := service.ValidateKey(context.Background(), "vl_live_testkey")

    // Assert — vérifier le résultat
    assert.Nil(t, result)
    assert.ErrorIs(t, err, ErrQuotaExceeded)
    mockRepo.AssertExpectations(t)
}
```

---

### Résumé visuel — ce qu'on cherche

```
Bonne fonction Veil                Mauvaise fonction Veil
───────────────────────────        ───────────────────────────
Nom explicite                      Nom abrégé (proc, hdl, tmp)
< 30 lignes                        > 80 lignes
1 responsabilité                   3-4 responsabilités mélangées
Early return                       Pyramide de if/else
Pas de magic numbers               "429", "100000" en dur
Pas de duplication                 Même logique copiée-collée
Testable (dépendances injectées)   Impossible à tester (new dans la fn)
```

---

## Variables d'environnement

```bash
# Serveur
PORT=8080
ENV=development                    # development | production

# Base de données
DATABASE_URL=postgres://...        # Supabase connection string

# Redis
REDIS_URL=redis://...              # Upstash Redis URL

# Providers (clés secrètes — JAMAIS en git)
DEEPSEEK_API_KEY=sk-...

# Stripe
STRIPE_SECRET_KEY=sk_live_...
STRIPE_WEBHOOK_SECRET=whsec_...

# Sécurité
API_KEY_SECRET=...                 # Salt pour hashing clés
JWT_SECRET=...                     # Pour dashboard auth
```

---

## Base de données — schéma MVP

```sql
-- users : comptes Veil
CREATE TABLE users (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email       TEXT UNIQUE NOT NULL,
    plan        TEXT NOT NULL DEFAULT 'free',
    stripe_id   TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- api_keys : clés d'accès (format vl_live_xxx)
CREATE TABLE api_keys (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    key_hash    TEXT UNIQUE NOT NULL,  -- SHA256, jamais la clé en clair
    name        TEXT NOT NULL,
    last_used   TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- requests : chaque requête pour analytics + billing
CREATE TABLE requests (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL REFERENCES users(id),
    provider     TEXT NOT NULL,        -- deepseek, groq...
    format       TEXT NOT NULL,        -- anthropic, openai, responses
    tokens_in    INT NOT NULL DEFAULT 0,
    tokens_out   INT NOT NULL DEFAULT 0,
    cost_usd     DECIMAL(10,6) NOT NULL DEFAULT 0,
    latency_ms   INT,
    status       TEXT NOT NULL,        -- success, error
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_requests_user_date ON requests(user_id, created_at DESC);

-- plans : définition des plans tarifaires
CREATE TABLE plans (
    id            TEXT PRIMARY KEY,   -- free, starter, pro, team
    price_usd     DECIMAL(10,2) NOT NULL,
    token_quota   BIGINT NOT NULL,
    max_api_keys  INT NOT NULL
);

INSERT INTO plans VALUES
    ('free',    0,    100000,   1),
    ('starter', 9,   2000000,   3),
    ('pro',     29, 10000000,  10),
    ('team',    99, 50000000,  50);
```

---

## Endpoints exposés

```
POST /v1/messages              ← Format Anthropic (Claude CLI)
POST /v1/chat/completions      ← Format OpenAI standard
POST /v1/responses             ← Format OpenAI Responses API (Codex)

GET  /health                   ← Health check (Dokploy)
GET  /v1/models                ← Liste des modèles (compatibilité)

POST /webhooks/stripe          ← Webhooks Stripe billing
```

---

## Format des clés API

```
Format   : vl_live_<32 bytes base64url>
Exemple  : vl_live_aB3xK9mN2pQrS7tU1vW4yZ8cD6eF0gH5
Stockage : SHA256 hash en base de données (jamais en clair)
Header   : Authorization: Bearer vl_live_xxx
```

---

## Business model

```
Plan      Prix      Tokens/mois    Clés API
────────────────────────────────────────────
free      $0        100k           1
starter   $9        2M             3
pro       $29       10M            10
team      $99       50M            50

Dépassement quota : $0.50 / 1M tokens supplémentaires
```

### Marges par provider

```
Provider    Coût achat     Prix vente     Marge
────────────────────────────────────────────────
DeepSeek    $0.14/1M       $0.40/1M       65%
Groq        $0.10/1M       $0.35/1M       71%
Mistral     $0.14/1M       $0.40/1M       65%
```

---

## Règles importantes

### Ce qu'on NE fait PAS

```
✗ Pas de ORM (pas de GORM) → sqlc + pgx uniquement
✗ Pas de fmt.Println       → zerolog uniquement
✗ Pas d'erreurs ignorées   → toujours gérer ou wrapper
✗ Pas de struct concrètes  → toujours injecter via interfaces
✗ Pas de .env en git       → .env.example uniquement
✗ Pas de clés en dur       → viper + variables d'env
✗ Pas de logique en main.go → main.go = wiring uniquement
```

### Ce qu'on fait TOUJOURS

```
✓ Context en premier paramètre de chaque fonction
✓ Erreurs wrappées avec fmt.Errorf("pkg.Func: %w", err)
✓ Tests pour chaque service (fichier _test.go)
✓ Interface avant implémentation
✓ Logs structurés JSON avec zerolog
✓ Timeouts sur tous les appels HTTP externes
✓ Migration SQL pour chaque changement de schéma
```

---

## Commandes utiles

```bash
# Développement local
docker-compose up -d          # Lance PG + Redis
go run cmd/server/main.go     # Lance le serveur

# Base de données
make migrate-up               # Applique les migrations
make migrate-down             # Rollback dernière migration
make sqlc                     # Génère le code SQL

# Tests
go test ./...                 # Tous les tests
go test ./internal/auth/...   # Tests d'un module
go test -race ./...           # Tests avec race detector

# Build
go build -o bin/veil cmd/server/main.go

# Lint
golangci-lint run
```

---

## Roadmap MVP (6 semaines)

```
Semaine 1-2  → Serveur Go : endpoints + translators DeepSeek
               PostgreSQL schema + Redis
               Auth middleware + token meter

Semaine 3    → Billing (Stripe) + quotas
               CLI Veil (Go + Cobra + BubbleTea)
               Docker Compose complet

Semaine 4    → Dashboard Next.js (stats + API keys)
               REPL interactif + slash commands
               Tests de charge (k6)

Semaine 5    → Deploy Hostinger + Dokploy
               Beta privée (20-50 users)

Semaine 6    → Fix bugs beta
               Landing page
               Launch Hacker News + ProductHunt
```

---

## Décisions architecturales (ADR)

### ADR-001 : Monolith Modulaire pour le MVP
**Décision** : Un seul binaire Go avec modules bien séparés par interfaces.
**Raison** : Rapidité de développement. Les interfaces permettent d'extraire
en microservices sans réécriture quand le besoin se présente.
**Révision** : À > 10k users simultanés.

### ADR-002 : sqlc plutôt que GORM
**Décision** : sqlc pour générer du code type-safe depuis les queries SQL.
**Raison** : Pas de magie, performances prévisibles, SQL explicite.
**Révision** : Jamais, sauf migration vers une autre DB.

### ADR-003 : DeepSeek uniquement au MVP
**Décision** : Un seul provider (DeepSeek) pour le MVP.
**Raison** : Simplicité. L'interface Provider permet d'ajouter Groq/Mistral
en Phase 2 sans toucher au reste du code.
**Révision** : Phase 2 (Groq, Mistral).

### ADR-004 : Redis Streams pour l'async
**Décision** : Redis Streams pour les events analytics et billing async.
**Raison** : Pas besoin de Kafka au MVP. Redis est déjà dans le stack.
**Révision** : À > 1M events/jour, évaluer Kafka.

### ADR-005 : Hostinger VPS + Dokploy
**Décision** : VPS Hostinger (2vCPU/4GB) + Dokploy pour le déploiement.
**Raison** : Coût prévisible ($15/mois), contrôle total, bonne latence
pour les utilisateurs africains. Dokploy = deploy automatique via GitHub.
**Révision** : À > 50k users, évaluer Kubernetes sur Hetzner.