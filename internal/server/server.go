package server

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/joekings2k/staypass-be/internal/auth"
	"github.com/joekings2k/staypass-be/internal/onboarding"
	"github.com/joekings2k/staypass-be/internal/platform/config"
	"github.com/joekings2k/staypass-be/internal/platform/logger"
	"github.com/joekings2k/staypass-be/internal/platform/token"
)

type Handlers struct {
	Onboarding *onboarding.Handler
	Auth       *auth.Handler
}

type Server struct {
	cfg      *config.Config
	handlers Handlers
	health   http.HandlerFunc
	maker    token.Maker
}

func New(cfg *config.Config, handlers Handlers, health http.HandlerFunc, maker token.Maker) *Server {
	return &Server{
		cfg:      cfg,
		handlers: handlers,
		health:   health,
		maker:    maker,
	}
}

func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(requestLogger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   s.cfg.CorsAllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Retry-After"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
	r.Use(middleware.Timeout(30 * time.Second))

	r.Get("/health", s.health)

	r.Route("/api/v1", func(r chi.Router) {
		s.handlers.Onboarding.RegisterRoutes(r)
	})

	return r
}

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log := slog.Default().With(
			"request_id", middleware.GetReqID(r.Context()),
			"method", r.Method,
			"path", r.URL.Path,
		)

		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		start := time.Now()
		log.Info("request started")

		next.ServeHTTP(ww, r.WithContext(logger.WithLogger(r.Context(), log)))

		log.Info("request completed",
			"status", ww.Status(),
			"bytes", ww.BytesWritten(),
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}
