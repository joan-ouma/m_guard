package handler

import (
	"net/http"

	"motoguard-backend/internal/domain"
)

type RouterConfig struct {
	AuthHandler         *AuthHandler
	BikeHandler         *BikeHandler
	HealthHandler       *HealthHandler
	AuthMiddleware      func(http.Handler) http.Handler
	RateLimitMiddleware func(http.Handler) http.Handler
}

func NewRouter(cfg RouterConfig) *http.ServeMux {
	mux := http.NewServeMux()

	if cfg.HealthHandler != nil {
		mux.Handle("GET /health", http.HandlerFunc(cfg.HealthHandler.Check))
		mux.Handle("GET /v1/health", http.HandlerFunc(cfg.HealthHandler.Check))
	}

	authMW := cfg.AuthMiddleware
	rateLimitMW := cfg.RateLimitMiddleware

	mux.Handle("POST /v1/users/register", rateLimitMW(http.HandlerFunc(cfg.AuthHandler.Register)))
	mux.Handle("POST /v1/users", rateLimitMW(http.HandlerFunc(cfg.AuthHandler.Register)))

	mux.Handle("POST /v1/sessions", rateLimitMW(http.HandlerFunc(cfg.AuthHandler.Login)))
	mux.Handle("PUT /v1/sessions/current", http.HandlerFunc(cfg.AuthHandler.Refresh))
	mux.Handle("DELETE /v1/sessions/current", http.HandlerFunc(cfg.AuthHandler.Logout))
	mux.Handle("GET /v1/sessions/current", authMW(http.HandlerFunc(cfg.AuthHandler.GetCurrent)))

	mux.Handle("POST /v1/password-resets", rateLimitMW(http.HandlerFunc(cfg.AuthHandler.RequestPasswordReset)))
	mux.Handle("PATCH /v1/password-resets/{token}", rateLimitMW(http.HandlerFunc(cfg.AuthHandler.ConfirmPasswordReset)))

	mux.Handle("GET /v1/bikes", authMW(http.HandlerFunc(cfg.BikeHandler.List)))
	mux.Handle("POST /v1/bikes", authMW(http.HandlerFunc(cfg.BikeHandler.Create)))
	mux.Handle("GET /v1/bikes/{id}", authMW(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.BikeHandler.GetByID(w, r, getBikeID(r))
	})))
	mux.Handle("PUT /v1/bikes/{id}", authMW(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.BikeHandler.Update(w, r, getBikeID(r))
	})))
	mux.Handle("PATCH /v1/bikes/{id}", authMW(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.BikeHandler.Update(w, r, getBikeID(r))
	})))
	mux.Handle("DELETE /v1/bikes/{id}", authMW(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.BikeHandler.Delete(w, r, getBikeID(r))
	})))
	mux.Handle("GET /v1/bikes/{id}/breadcrumbs", authMW(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.BikeHandler.GetBreadcrumbs(w, r, getBikeID(r))
	})))
	mux.Handle("GET /v1/bikes/{id}/events", authMW(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.BikeHandler.GetEvents(w, r, getBikeID(r))
	})))
	mux.Handle("POST /v1/bikes/{id}/immobilize", authMW(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.BikeHandler.Immobilize(w, r, getBikeID(r))
	})))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		domain.NewNotFoundError("Resource not found", r.URL.Path).WriteJSON(w)
	})

	return mux
}
