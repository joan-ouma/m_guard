package mailer

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMailjetMailer_SendPasswordResetEmail(t *testing.T) {
	apiKey := "test-api-key"
	secretKey := "test-secret-key"
	senderEmail := "no-reply@jullie.tech"
	replyToEmail := "hello@jullie.tech"
	senderName := "MotoGuard"

	var receivedReq *http.Request
	var receivedPayload mailjetPayload

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedReq = r
		if err := json.NewDecoder(r.Body).Decode(&receivedPayload); err != nil {
			t.Errorf("failed to decode payload: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		resp := mailjetResponse{
			Messages: []mailjetResponseItem{
				{Status: "success"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	mailer := NewMailjetMailer(apiKey, secretKey, senderEmail, senderName, replyToEmail)
	mailer.SetEndpoint(server.URL)
	mailer.SetHTTPClient(server.Client())

	ctx := context.Background()
	toEmail := "user@example.com"
	toName := "Alice"
	resetURL := "http://localhost:4200/reset-password/sample-reset-token"

	err := mailer.SendPasswordResetEmail(ctx, toEmail, toName, resetURL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	user, pass, ok := receivedReq.BasicAuth()
	if !ok || user != apiKey || pass != secretKey {
		t.Errorf("expected basic auth (%s, %s), got (%s, %s)", apiKey, secretKey, user, pass)
	}

	if len(receivedPayload.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(receivedPayload.Messages))
	}

	msg := receivedPayload.Messages[0]
	if msg.From.Email != senderEmail || msg.From.Name != senderName {
		t.Errorf("expected from %s <%s>, got %s <%s>", senderName, senderEmail, msg.From.Name, msg.From.Email)
	}
	if msg.ReplyTo == nil || msg.ReplyTo.Email != replyToEmail {
		t.Errorf("expected reply-to %s, got %+v", replyToEmail, msg.ReplyTo)
	}
	if !strings.Contains(msg.TextPart, resetURL) {
		t.Errorf("expected reset URL in text part, got: %s", msg.TextPart)
	}
	if !strings.Contains(msg.HTMLPart, resetURL) {
		t.Errorf("expected reset URL in HTML part, got: %s", msg.HTMLPart)
	}
}

func TestMailjetMailer_SendGeofenceBreachAlertEmail(t *testing.T) {
	var receivedPayload mailjetPayload
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&receivedPayload)
		resp := mailjetResponse{Messages: []mailjetResponseItem{{Status: "success"}}}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	mailer := NewMailjetMailer("k", "s", "no-reply@jullie.tech", "MotoGuard", "hello@jullie.tech")
	mailer.SetEndpoint(server.URL)
	mailer.SetHTTPClient(server.Client())

	data := GeofenceAlertData{
		BikeRegistration: "KMCB 241A",
		BikeModel:        "Honda Ace 125",
		GeofenceName:     "Nairobi CBD Perimeter",
		BreachType:       "Boundary Exit",
		Timestamp:        "2026-09-29 23:45 UTC",
		Address:          "Mombasa Road, Nairobi",
		SpeedKMH:         "45",
		DashboardURL:     "http://localhost:4200/dashboard",
	}

	err := mailer.SendGeofenceBreachAlertEmail(context.Background(), "admin@example.com", "Admin", data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(receivedPayload.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(receivedPayload.Messages))
	}
	msg := receivedPayload.Messages[0]
	if !strings.Contains(msg.Subject, "Geofence Breach: KMCB 241A") {
		t.Errorf("unexpected subject: %s", msg.Subject)
	}
	if !strings.Contains(msg.HTMLPart, "KMCB 241A") || !strings.Contains(msg.HTMLPart, "Nairobi CBD Perimeter") {
		t.Errorf("expected bike and geofence in HTML part")
	}
}

func TestMailjetMailer_SendBikeTouchedAlertEmail(t *testing.T) {
	var receivedPayload mailjetPayload
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&receivedPayload)
		resp := mailjetResponse{Messages: []mailjetResponseItem{{Status: "success"}}}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	mailer := NewMailjetMailer("k", "s", "no-reply@jullie.tech", "MotoGuard", "hello@jullie.tech")
	mailer.SetEndpoint(server.URL)
	mailer.SetHTTPClient(server.Client())

	data := BikeTouchedAlertData{
		BikeRegistration: "KMCB 241A",
		BikeModel:        "Honda Ace 125",
		Timestamp:        "2026-09-29 23:45 UTC",
		Address:          "Kenyatta Avenue, Nairobi",
		BatteryPercent:   88,
		DashboardURL:     "http://localhost:4200/dashboard",
	}

	err := mailer.SendBikeTouchedAlertEmail(context.Background(), "admin@example.com", "Admin", data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(receivedPayload.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(receivedPayload.Messages))
	}
	msg := receivedPayload.Messages[0]
	if !strings.Contains(msg.Subject, "Unauthorized Movement") {
		t.Errorf("unexpected subject: %s", msg.Subject)
	}
	if !strings.Contains(msg.HTMLPart, "Movement / Tamper Sensor Triggered") {
		t.Errorf("expected sensor trigger in HTML part")
	}
}
