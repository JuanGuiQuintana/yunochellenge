package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/juanatsap/chargeback-api/internal/api"
	"github.com/juanatsap/chargeback-api/internal/api/handler"
	"github.com/juanatsap/chargeback-api/internal/config"
	"github.com/juanatsap/chargeback-api/internal/db"
	"github.com/juanatsap/chargeback-api/internal/migration"
	proc "github.com/juanatsap/chargeback-api/internal/processor"
	"github.com/juanatsap/chargeback-api/internal/processor/acquireco"
	"github.com/juanatsap/chargeback-api/internal/processor/globalpay"
	"github.com/juanatsap/chargeback-api/internal/processor/payflow"
	"github.com/juanatsap/chargeback-api/internal/repository"
	"github.com/juanatsap/chargeback-api/internal/scoring"
	"github.com/juanatsap/chargeback-api/internal/service"
)

func main() {
	if os.Getenv("APP_ENV") != "production" {
		if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
			slog.Warn("could not load .env file", "error", err)
		}
	}

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	logger := buildLogger(cfg.App.LogLevel)
	slog.SetDefault(logger)

	ctx := context.Background()

	pool, err := db.New(ctx, cfg.Database)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	slog.Info("connected to database")

	if cfg.App.RunMigrations {
		if err := migration.Run(cfg.Database.URL, "./migrations"); err != nil {
			slog.Error("failed to run migrations", "error", err)
			os.Exit(1)
		}
		slog.Info("migrations applied")
	}

	// Build repositories
	cbRepo := repository.NewChargebackRepository(pool)
	mRepo := repository.NewMerchantRepository(pool)
	chRepo := repository.NewCardholderRepository(pool)
	fxRepo := repository.NewFxRateRepository(pool)
	rlRepo := repository.NewRateLimitRepository(pool)
	procRepo := repository.NewProcessorRepository(pool)
	rcRepo := repository.NewReasonCodeRepository(pool)

	// Build processor adapters
	adapters := map[string]proc.ProcessorAdapter{
		"acquireco": acquireco.New(),
		"payflow":   payflow.New(),
		"globalpay": globalpay.New(),
	}

	// Build services
	scoringEngine := scoring.NewScoringEngine()
	enrichmentSvc := service.NewEnrichmentService(fxRepo)
	patternSvc := service.NewPatternDetectionService(pool)

	ingestSvc := service.NewIngestService(
		adapters,
		cbRepo, mRepo, chRepo, fxRepo, rlRepo, procRepo, rcRepo,
		scoringEngine, enrichmentSvc, patternSvc,
	)
	querySvc := service.NewQueryService(cbRepo, mRepo, procRepo)

	// Build handlers
	ingestH := handler.NewIngestHandler(ingestSvc)
	chargebackH := handler.NewChargebackHandler(querySvc)
	merchantH := handler.NewMerchantHandler(querySvc)

	// Build router and server
	router := api.NewRouter(ingestH, chargebackH, merchantH)
	srv := api.NewServer(
		cfg.Server.Port,
		router,
		cfg.Server.ReadTimeout,
		cfg.Server.WriteTimeout,
		cfg.Server.IdleTimeout,
	)

	shutdownErr := make(chan error, 1)
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit

		slog.Info("shutdown signal received")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		shutdownErr <- srv.Shutdown(shutdownCtx)
	}()

	slog.Info("server starting", "port", cfg.Server.Port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}

	if err := <-shutdownErr; err != nil {
		slog.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}
	slog.Info("server stopped")
}

func buildLogger(level string) *slog.Logger {
	var logLevel slog.Level
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))
}
