// @title          Veil API
// @version        1.0
// @description    Gateway universel : utilise tes outils habituels (Claude CLI, Cursor, Aider) avec des modèles moins chers en arrière-plan (DeepSeek, Groq, Mistral). Garde ton workflow. Réduis ta facture API de 90%.
// @contact.name   Support Veil
// @contact.email  support@veil.dev
// @host           localhost:3000
// @BasePath       /
// @schemes        http https
// @securityDefinitions.apikey DashboardAuth
// @in header
// @name Authorization
// @description JWT Dashboard — pour les endpoints /api/*

// @securityDefinitions.apikey VeilAPIKey
// @in header
// @name Authorization
// @description Clé Veil vl_live_xxx — pour les endpoints /v1/*
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	fiberswagger "github.com/gofiber/swagger"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/thatsbass/veil/docs"
	"github.com/thatsbass/veil/internal/analytics"
	internalapi "github.com/thatsbass/veil/internal/api"
	apikeys "github.com/thatsbass/veil/internal/api/keys"
	apibilling "github.com/thatsbass/veil/internal/api/billing"
	apiusage "github.com/thatsbass/veil/internal/api/usage"
	"github.com/thatsbass/veil/internal/auth"
	clerkprovider "github.com/thatsbass/veil/internal/auth/clerk"
	"github.com/thatsbass/veil/internal/billing"
	"github.com/thatsbass/veil/internal/config"
	"github.com/thatsbass/veil/internal/gateway"
	"github.com/thatsbass/veil/internal/mailer"
	mailernoop "github.com/thatsbass/veil/internal/mailer/noop"
	mailerresend "github.com/thatsbass/veil/internal/mailer/resend"
	"github.com/thatsbass/veil/internal/migrate"
	"github.com/thatsbass/veil/internal/provider"
	"github.com/thatsbass/veil/internal/translator"
)

func main() {
	log.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}
	if !cfg.IsProduction() {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	}
	docs.SwaggerInfo.Host = cfg.Host

	if err := migrate.Run(cfg.DatabaseURL); err != nil {
		log.Fatal().Err(err).Msg("migrations failed")
	}
	log.Info().Msg("migrations applied")

	db := mustConnectDB(cfg.DatabaseURL)
	defer db.Close()

	rdb := mustConnectRedis(cfg.RedisURL)
	defer rdb.Close()

	// Mailer — noop si RESEND_API_KEY absent (dev local)
	var mailSvc mailer.Mailer
	if cfg.ResendAPIKey != "" {
		mailSvc = mailerresend.NewProvider(cfg.ResendAPIKey)
	} else {
		mailSvc = mailernoop.NewProvider()
		log.Warn().Msg("RESEND_API_KEY not set, using noop mailer")
	}
	_ = mailSvc // sera injecté dans les handlers au fil des features

	// Providers
	deepSeek := provider.NewDeepSeek(cfg.DeepSeekAPIKey)

	// Auth
	authRepo := auth.NewCachedRepository(db, rdb)
	quotaChecker := auth.NewRedisQuotaChecker(rdb)
	authSvc := auth.NewAuthService(authRepo, quotaChecker)

	// Billing
	meter := billing.NewMeter(rdb)
	quotaMgr := billing.NewQuotaManager(rdb)
	billingSvc := billing.NewBillingService(meter, quotaMgr)
	stripeHandler := billing.NewStripeHandler(cfg.StripeWebhookSecret, log.Logger)

	// Analytics
	analyticsSvc := analytics.NewRedisTracker(rdb)

	// Translators
	anthropicTranslator := translator.NewAnthropic()
	openAITranslator := translator.NewOpenAI()
	responsesTranslator := translator.NewResponses()

	// Gateway handler
	h := gateway.NewHandler(
		anthropicTranslator,
		openAITranslator,
		responsesTranslator,
		deepSeek,
		billingSvc,
		analyticsSvc,
		log.Logger,
	)

	app := fiber.New(fiber.Config{
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
	})
	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New())

	// Swagger UI — accessible à http://localhost:3000/swagger/
	app.Get("/swagger/*", fiberswagger.HandlerDefault)

	// Public routes
	app.Get("/health", h.HandleHealth)
	app.Post("/webhooks/stripe", stripeHandler.HandleWebhook)

	// Dashboard handlers (/api/*)
	userResolver := internalapi.NewPGUserResolver(db)
	keysHandler := apikeys.NewHandler(userResolver, apikeys.NewPGKeyRepository(db))
	usageHandler := apiusage.NewHandler(userResolver, billingSvc, apiusage.NewPGUsageRepository(db))
	billingHandler := apibilling.NewHandler(userResolver, apibilling.NewPGPlanRepository(db))

	// /api/* — dashboard Next.js (Clerk JWT)
	var authProvider auth.AuthProvider = clerkprovider.NewProvider(cfg.ClerkSecretKey)
	api := app.Group("/api", auth.DashboardAuthMiddleware(authProvider))
	api.Post("/keys", keysHandler.CreateAPIKey)
	api.Get("/keys", keysHandler.ListAPIKeys)
	api.Delete("/keys/:id", keysHandler.RevokeAPIKey)
	api.Get("/usage", usageHandler.GetCurrentUsage)
	api.Get("/usage/history", usageHandler.GetUsageHistory)
	api.Get("/billing/plan", billingHandler.GetCurrentPlan)
	api.Post("/billing/upgrade", billingHandler.UpgradePlan)

	// /v1/* — clients LLM (clé vl_live_xxx)
	protected := app.Group("/v1", auth.APIKeyMiddleware(authSvc))
	protected.Get("/models", h.HandleModels)
	protected.Post("/messages", h.HandleMessages)
	protected.Post("/chat/completions", h.HandleChatCompletions)
	protected.Post("/responses", h.HandleResponses)

	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Info().Str("addr", addr).Msg("veil server starting")

	go func() {
		if err := app.Listen(addr); err != nil {
			log.Fatal().Err(err).Msg("server error")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("shutting down")
	if err := app.ShutdownWithTimeout(10 * time.Second); err != nil {
		log.Error().Err(err).Msg("shutdown error")
	}
}

func mustConnectDB(dsn string) *pgxpool.Pool {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to postgres")
	}
	if err := pool.Ping(ctx); err != nil {
		log.Fatal().Err(err).Msg("failed to ping postgres")
	}
	log.Info().Msg("postgres connected")
	return pool
}

func mustConnectRedis(url string) *redis.Client {
	opts, err := redis.ParseURL(url)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to parse redis url")
	}

	rdb := redis.NewClient(opts)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatal().Err(err).Msg("failed to ping redis")
	}
	log.Info().Msg("redis connected")
	return rdb
}
