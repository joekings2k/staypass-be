package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joekings2k/staypass-be/internal/auth"
	"github.com/joekings2k/staypass-be/internal/membership"
	"github.com/joekings2k/staypass-be/internal/onboarding"
	"github.com/joekings2k/staypass-be/internal/platform/config"
	"github.com/joekings2k/staypass-be/internal/platform/database"
	dbgen "github.com/joekings2k/staypass-be/internal/platform/db/gen"
	"github.com/joekings2k/staypass-be/internal/platform/hash"
	"github.com/joekings2k/staypass-be/internal/platform/logger"
	"github.com/joekings2k/staypass-be/internal/platform/token"
	"github.com/joekings2k/staypass-be/internal/server"
	"github.com/joekings2k/staypass-be/internal/user"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}
	ctx := context.Background()

	log := logger.NewLogger(cfg)
	slog.SetDefault(log)

	pool, err := database.New(ctx, cfg.DatabaseURL)

	if err != nil {
		log.Error("failed to create database pool", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	q := dbgen.New(pool)
	tx := database.NewTransactionRunner(pool)

	tokenMaker, err := token.NewPasetoMaker(cfg.TokenSymmetricKey)
	if err != nil {
		log.Error("failed to create token maker", "error", err)
		os.Exit(1)
	}
	hasher := hash.NewBcryptHasher()
	authRepository := auth.NewPgAuthRepository(q)
	userRepository := user.NewPgUserRepository(q)
	membershipRepository := membership.NewMembershipRepository(q)

	authService := auth.NewAuthService(authRepository, userRepository, membershipRepository, cfg, hasher, tokenMaker, tx)
	authHandler := auth.NewHandlerService(authService)

	onboardingService := onboarding.NewOnboardingService(tx, hasher)
	onboardingHandler := onboarding.NewHandler(onboardingService)

	appServer := server.New(cfg, server.Handlers{
		Onboarding: onboardingHandler,
		Auth:       authHandler,
	}, handleHealth(pool), tokenMaker)

	srv := &http.Server{
		Addr:              ":" + cfg.AppPort,
		Handler:           appServer.Router(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		slog.Info("server starting", "port", cfg.AppPort, "env", cfg.AppEnv)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	slog.Info("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("shutdown failed", "error", err)
	}
}

func handleHealth(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := pool.Ping(ctx); err != nil {
			slog.Error("db ping failed", "error", err)
			http.Error(w, "database connection failed", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status": "ok",
			"time":   time.Now().UTC().Format(time.RFC3339),
		})
	}
}
