package handler

import (
	"net/http"

	"motoguard-backend/internal/domain"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

type RouterConfig struct {
	AuthHandler         *AuthHandler
	BikeHandler         *BikeHandler
	HealthHandler       *HealthHandler
	NotificationHandler *NotificationHandler
	SSEHandler          *SSEHandler
	USSDHandler         *USSDHandler
	AuthMiddleware      func(http.Handler) http.Handler
	RateLimitMiddleware func(http.Handler) http.Handler
}

func NewRouter(cfg RouterConfig) *chi.Mux {
	r := chi.NewRouter()

	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Recoverer)

	r.Get("/health", cfg.HealthHandler.Check)
	r.Get("/v1/health", cfg.HealthHandler.Check)

	r.Mount("/v1/users", cfg.AuthHandler.UserRoutes(cfg.RateLimitMiddleware))
	r.Mount("/v1/sessions", cfg.AuthHandler.SessionRoutes(cfg.RateLimitMiddleware, cfg.AuthMiddleware))
	r.Mount("/v1/password-resets", cfg.AuthHandler.PasswordResetRoutes(cfg.RateLimitMiddleware))
	r.Mount("/v1/bikes", cfg.BikeHandler.Routes(cfg.AuthMiddleware, cfg.SSEHandler.Stream))
	if cfg.NotificationHandler != nil {
		r.Mount("/v1/notifications", cfg.NotificationHandler.Routes(cfg.AuthMiddleware))
	}

	if cfg.USSDHandler != nil {
		r.Mount("/v1/ussd", cfg.USSDHandler.Routes())
		r.Mount("/ussd", cfg.USSDHandler.Routes())
		r.Mount("/v1/sms", cfg.USSDHandler.SMSRoutes())
		r.Mount("/sms", cfg.USSDHandler.SMSRoutes())
		r.Get("/simulator", cfg.USSDHandler.SimulatorPage)
		r.Get("/v1/simulator", cfg.USSDHandler.SimulatorPage)
	}

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		domain.NewNotFoundError("Resource not found", r.URL.Path).WriteJSON(w)
	})

	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		domain.NewMethodNotAllowedError("Method not allowed", r.URL.Path).WriteJSON(w)
	})

	return r
}
