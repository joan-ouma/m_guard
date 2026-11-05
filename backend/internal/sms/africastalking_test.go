package sms

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSendSMS(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if got := r.Header.Get("apiKey"); got != "test-key" {
			t.Errorf("expected API key header, got %q", got)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if got := r.Form.Get("username"); got != "test-user" {
			t.Errorf("expected username, got %q", got)
		}
		if got := r.Form.Get("to"); got != "+254712345678" {
			t.Errorf("expected recipient, got %q", got)
		}
		if got := r.Form.Get("from"); got != "20384" {
			t.Errorf("expected sender shortcode, got %q", got)
		}
		if got := r.Form.Get("message"); got != "Police station: 0721 334 999" {
			t.Errorf("unexpected message: %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"SMSMessageData":{"Recipients":[{"statusCode":101,"status":"Success"}]}}`))
	}))
	defer server.Close()

	client := NewClient("test-user", "test-key", "20384")
	client.endpoint = server.URL
	if err := client.SendSMS(context.Background(), "+254712345678", "Police station: 0721 334 999"); err != nil {
		t.Fatalf("SendSMS returned error: %v", err)
	}
}

func TestSendSMSRejectsUnsuccessfulProviderResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"SMSMessageData":{"Recipients":[{"statusCode":402,"status":"InvalidSender"}]}}`))
	}))
	defer server.Close()

	client := NewClient("test-user", "test-key", "")
	client.endpoint = server.URL
	err := client.SendSMS(context.Background(), "+254712345678", "test")
	if err == nil || !strings.Contains(err.Error(), "not accepted") {
		t.Fatalf("expected provider rejection error, got %v", err)
	}
}

func TestSendSMSRequiresCredentials(t *testing.T) {
	client := NewClient("", "", "")
	if err := client.SendSMS(context.Background(), "+254712345678", "test"); err == nil {
		t.Fatal("expected missing-credentials error")
	}
}
