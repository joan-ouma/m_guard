package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimitMiddleware(t *testing.T) {
	mw := RateLimitMiddleware(3, 100*time.Millisecond)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest(http.MethodPost, "/v1/password-resets", nil)
	req.RemoteAddr = "192.168.1.100:1234"

	// First 3 requests should succeed
	for i := 0; i < 3; i++ {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("request %d expected status 200, got %d", i+1, rr.Code)
		}
	}

	// 4th request should return 429 Too Many Requests
	rr4 := httptest.NewRecorder()
	handler.ServeHTTP(rr4, req)
	if rr4.Code != http.StatusTooManyRequests {
		t.Fatalf("request 4 expected status 429, got %d", rr4.Code)
	}

	// Different IP should still succeed
	reqOther := httptest.NewRequest(http.MethodPost, "/v1/password-resets", nil)
	reqOther.RemoteAddr = "192.168.1.101:1234"
	rrOther := httptest.NewRecorder()
	handler.ServeHTTP(rrOther, reqOther)
	if rrOther.Code != http.StatusOK {
		t.Fatalf("other IP request expected status 200, got %d", rrOther.Code)
	}

	// Wait for window to expire
	time.Sleep(120 * time.Millisecond)

	// Should succeed again
	rr5 := httptest.NewRecorder()
	handler.ServeHTTP(rr5, req)
	if rr5.Code != http.StatusOK {
		t.Fatalf("request after window expiry expected status 200, got %d", rr5.Code)
	}
}
