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

	router := NewRouter(RouterConfig{
		AuthHandler:         &AuthHandler{},
		BikeHandler:         &BikeHandler{},
		AuthMiddleware:      passthroughMW,
		RateLimitMiddleware: passthroughMW,
	})

	// Verify route exists (method not allowed or handled rather than 404)
	req := httptest.NewRequest(http.MethodOptions, "/v1/sessions", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Since POST is registered, OPTIONS or other methods will get 405 (Method Not Allowed) from Go 1.22+ ServeMux
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405 for unhandled method on defined route, got %d", rr.Code)
	}

	// Non-existent route should return 404
	req404 := httptest.NewRequest(http.MethodGet, "/v1/nonexistent", nil)
	rr404 := httptest.NewRecorder()
	router.ServeHTTP(rr404, req404)

	if rr404.Code != http.StatusNotFound {
		t.Fatalf("expected status 404 for undefined route, got %d", rr404.Code)
	}
}
