package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"motoguard-backend/internal/service"
)

type routerSMSSender struct{}

func (routerSMSSender) SendSMS(context.Context, string, string) error { return nil }

func TestNewRouter(t *testing.T) {
	passthroughMW := func(next http.Handler) http.Handler {
		return next
	}

	healthH := NewHealthHandler(nil)

	router := NewRouter(RouterConfig{
		AuthHandler:         &AuthHandler{},
		BikeHandler:         &BikeHandler{},
		HealthHandler:       healthH,
		SSEHandler:          &SSEHandler{},
		SMSHandler:          NewSMSHandler(service.NewSMSService(routerSMSSender{}, "kamilimu", "ignition")),
		AuthMiddleware:      passthroughMW,
		RateLimitMiddleware: passthroughMW,
	})

	reqHealth := httptest.NewRequest(http.MethodGet, "/health", nil)
	rrHealth := httptest.NewRecorder()
	router.ServeHTTP(rrHealth, reqHealth)

	if rrHealth.Code != http.StatusOK {
		t.Fatalf("expected status 200 for /health, got %d", rrHealth.Code)
	}

	callbackForm := url.Values{
		"from": {" +254712345678 "},
		"text": {"kamilimu ignition police Rongai"},
	}
	reqSMS := httptest.NewRequest(http.MethodPost, "/sms/africastalking", strings.NewReader(callbackForm.Encode()))
	reqSMS.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rrSMS := httptest.NewRecorder()
	router.ServeHTTP(rrSMS, reqSMS)
	if rrSMS.Code != http.StatusOK {
		t.Fatalf("expected status 200 for SMS callback, got %d", rrSMS.Code)
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
