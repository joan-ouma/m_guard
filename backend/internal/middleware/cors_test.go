package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSMiddleware(t *testing.T) {
	handler := CORSMiddleware([]string{"http://localhost:4200"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Preflight OPTIONS with allowed origin
	reqOptions := httptest.NewRequest(http.MethodOptions, "/v1/bikes", nil)
	reqOptions.Header.Set("Origin", "http://localhost:4200")
	rrOptions := httptest.NewRecorder()
	handler.ServeHTTP(rrOptions, reqOptions)

	if rrOptions.Code != http.StatusNoContent {
		t.Fatalf("expected 204 No Content on OPTIONS, got %d", rrOptions.Code)
	}
	if origin := rrOptions.Header().Get("Access-Control-Allow-Origin"); origin != "http://localhost:4200" {
		t.Fatalf("expected CORS origin http://localhost:4200, got %s", origin)
	}

	// Request with disallowed origin
	reqDisallowed := httptest.NewRequest(http.MethodGet, "/v1/bikes", nil)
	reqDisallowed.Header.Set("Origin", "http://evil.com")
	rrDisallowed := httptest.NewRecorder()
	handler.ServeHTTP(rrDisallowed, reqDisallowed)

	if origin := rrDisallowed.Header().Get("Access-Control-Allow-Origin"); origin != "" {
		t.Fatalf("expected no CORS origin for disallowed origin, got %s", origin)
	}

	// Test wildcard domain *.jullie.tech with mxguard.jullie.tech
	wildcardHandler := CORSMiddleware([]string{"*.jullie.tech"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	reqWildcard := httptest.NewRequest(http.MethodOptions, "/v1/sessions", nil)
	reqWildcard.Header.Set("Origin", "https://mxguard.jullie.tech")
	reqWildcard.Header.Set("Access-Control-Request-Headers", "content-type, authorization")
	rrWildcard := httptest.NewRecorder()
	wildcardHandler.ServeHTTP(rrWildcard, reqWildcard)

	if rrWildcard.Code != http.StatusNoContent {
		t.Fatalf("expected 204 No Content on wildcard OPTIONS, got %d", rrWildcard.Code)
	}
	if origin := rrWildcard.Header().Get("Access-Control-Allow-Origin"); origin != "https://mxguard.jullie.tech" {
		t.Fatalf("expected CORS origin https://mxguard.jullie.tech, got %s", origin)
	}
	if allowHeaders := rrWildcard.Header().Get("Access-Control-Allow-Headers"); allowHeaders != "content-type, authorization" {
		t.Fatalf("expected echoed headers, got %s", allowHeaders)
	}
}
