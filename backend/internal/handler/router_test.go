package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewRouter(t *testing.T) {
	passthroughMW := func(next http.Handler) http.Handler {
		return next
	}

	healthH := NewHealthHandler(nil)

	router := NewRouter(RouterConfig{
		AuthHandler:         &AuthHandler{},
		BikeHandler:         &BikeHandler{},
		HealthHandler:       healthH,
		AuthMiddleware:      passthroughMW,
		RateLimitMiddleware: passthroughMW,
	})

	reqHealth := httptest.NewRequest(http.MethodGet, "/health", nil)
	rrHealth := httptest.NewRecorder()
	router.ServeHTTP(rrHealth, reqHealth)

	if rrHealth.Code != http.StatusOK {
		t.Fatalf("expected status 200 for /health, got %d", rrHealth.Code)
	}

	reqV1Health := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	rrV1Health := httptest.NewRecorder()
	router.ServeHTTP(rrV1Health, reqV1Health)

	if rrV1Health.Code != http.StatusOK {
		t.Fatalf("expected status 200 for /v1/health, got %d", rrV1Health.Code)
	}

	req := httptest.NewRequest(http.MethodOptions, "/v1/sessions", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405 for unhandled method on defined route, got %d", rr.Code)
	}

	req404 := httptest.NewRequest(http.MethodGet, "/v1/nonexistent", nil)
	rr404 := httptest.NewRecorder()
	router.ServeHTTP(rr404, req404)

	if rr404.Code != http.StatusNotFound {
		t.Fatalf("expected status 404 for undefined route, got %d", rr404.Code)
	}
}
