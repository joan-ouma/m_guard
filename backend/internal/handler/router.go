package handler

import (
	"net/http"
)

// RouterConfig contains dependencies required to assemble the HTTP router.
type RouterConfig struct {
	AuthHandler         *AuthHandler
	BikeHandler         *BikeHandler
	AuthMiddleware      func(http.Handler) http.Handler
	RateLimitMiddleware func(http.Handler) http.Handler
}

// NewRouter constructs and registers all API routes on a standard http.ServeMux.
func NewRouter(cfg RouterConfig) *http.ServeMux {
	mux := http.NewServeMux()

	authMW := cfg.AuthMiddleware
	rateLimitMW := cfg.RateLimitMiddleware

	// Authentication & Sessions
	mux.Handle("POST /v1/sessions", rateLimitMW(http.HandlerFunc(cfg.AuthHandler.Login)))
	mux.Handle("PUT /v1/sessions/current", http.HandlerFunc(cfg.AuthHandler.Refresh))
	mux.Handle("DELETE /v1/sessions/current", http.HandlerFunc(cfg.AuthHandler.Logout))
	mux.Handle("GET /v1/sessions/current", authMW(http.HandlerFunc(cfg.AuthHandler.GetCurrent)))

	// Password Recovery
	mux.Handle("POST /v1/password-resets", rateLimitMW(http.HandlerFunc(cfg.AuthHandler.RequestPasswordReset)))
	mux.Handle("PATCH /v1/password-resets/{token}", rateLimitMW(http.HandlerFunc(cfg.AuthHandler.ConfirmPasswordReset)))

	// Fleet Management
	mux.Handle("GET /v1/bikes", authMW(http.HandlerFunc(cfg.BikeHandler.List)))
	mux.Handle("POST /v1/bikes", authMW(http.HandlerFunc(cfg.BikeHandler.Create)))
	mux.Handle("GET /v1/bikes/{id}", authMW(http.HandlerFunc(cfg.BikeHandler.GetByID)))
	mux.Handle("PUT /v1/bikes/{id}", authMW(http.HandlerFunc(cfg.BikeHandler.Update)))
	mux.Handle("PATCH /v1/bikes/{id}", authMW(http.HandlerFunc(cfg.BikeHandler.Update)))
	mux.Handle("DELETE /v1/bikes/{id}", authMW(http.HandlerFunc(cfg.BikeHandler.Delete)))
	mux.Handle("GET /v1/bikes/{id}/breadcrumbs", authMW(http.HandlerFunc(cfg.BikeHandler.GetBreadcrumbs)))
	mux.Handle("GET /v1/bikes/{id}/events", authMW(http.HandlerFunc(cfg.BikeHandler.GetEvents)))
	mux.Handle("POST /v1/bikes/{id}/immobilize", authMW(http.HandlerFunc(cfg.BikeHandler.Immobilize)))

	return mux
}
