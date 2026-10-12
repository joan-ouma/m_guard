package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"motoguard-backend/internal/security"
)

func TestAuthMiddleware(t *testing.T) {
	jwtManager := security.NewJWTManager("test-secret-key-32-bytes-long!!!!", time.Hour)

	var capturedUserID string
	handler := AuthMiddleware(jwtManager)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := GetClaims(r.Context())
		if ok && claims != nil {
			capturedUserID = claims.UserID
		}
		w.WriteHeader(http.StatusOK)
	}))

	// Case 1: Missing auth header
	reqNoAuth := httptest.NewRequest(http.MethodGet, "/v1/sessions/current", nil)
	rrNoAuth := httptest.NewRecorder()
	handler.ServeHTTP(rrNoAuth, reqNoAuth)
	if rrNoAuth.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401 without auth, got %d", rrNoAuth.Code)
	}

	// Case 2: Invalid token
	reqInvalid := httptest.NewRequest(http.MethodGet, "/v1/sessions/current", nil)
	reqInvalid.Header.Set("Authorization", "Bearer invalid-token")
	rrInvalid := httptest.NewRecorder()
	handler.ServeHTTP(rrInvalid, reqInvalid)
	if rrInvalid.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401 with invalid token, got %d", rrInvalid.Code)
	}

	// Case 3: Valid token
	validToken, err := jwtManager.GenerateAccessToken("user-123")
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	reqValid := httptest.NewRequest(http.MethodGet, "/v1/sessions/current", nil)
	reqValid.Header.Set("Authorization", "Bearer "+validToken)
	rrValid := httptest.NewRecorder()
	handler.ServeHTTP(rrValid, reqValid)

	if rrValid.Code != http.StatusOK {
		t.Fatalf("expected status 200 with valid token, got %d", rrValid.Code)
	}
	if capturedUserID != "user-123" {
		t.Fatalf("expected captured user id user-123, got %s", capturedUserID)
	}
}
