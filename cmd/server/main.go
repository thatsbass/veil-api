// @title          Veil API
// @version        1.0
// @description    Gateway universel : utilise tes outils habituels (Claude CLI, Cursor, Aider) avec des modèles moins chers en arrière-plan (DeepSeek, Groq, Mistral). Garde ton workflow. Réduis ta facture API de 90%.
// @contact.name   Support Veil
// @contact.email  support@veil.dev
// @host           localhost:3000
// @BasePath       /
// @schemes        http https
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Clé API Veil. Cliquer sur "Authorize" et entrer : Bearer vl_live_xxx
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

	"github.com/thatsbass/veil/internal/analytics"
	"github.com/thatsbass/veil/internal/auth"
	"github.com/thatsbass/veil/internal/billing"
	"github.com/thatsbass/veil/internal/config"
	"github.com/thatsbass/veil/internal/gateway"
	"github.com/thatsbass/veil/internal/migrate"
	"github.com/thatsbass/veil/internal/provider"
	"github.com/thatsbass/veil/internal/translator"
	_ "github.com/thatsbass/veil/docs"
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

	if err := migrate.Run(cfg.DatabaseURL); err != nil {
		log.Fatal().Err(err).Msg("migrations failed")
	}
	log.Info().Msg("migrations applied")

	db := mustConnectDB(cfg.DatabaseURL)
	defer db.Close()

	rdb := mustConnectRedis(cfg.RedisURL)
	defer rdb.Close()

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

	// Authenticated routes
	protected := app.Group("/v1", auth.Middleware(authSvc))
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
