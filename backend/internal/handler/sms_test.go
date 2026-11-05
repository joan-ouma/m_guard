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

type smsHandlerSender struct {
	to      string
	message string
}

func (s *smsHandlerSender) SendSMS(_ context.Context, to, message string) error {
	s.to = to
	s.message = message
	return nil
}

func TestHandleAfricaTalkingFormCallback(t *testing.T) {
	sender := &smsHandlerSender{}
	smsService := service.NewSMSService(sender, "kamilimu", "ignition")
	handler := NewSMSHandler(smsService)

	form := url.Values{
		"from": {" +254712345678 "},
		"text": {"kamilimu ignition police Rongai"},
	}
	req := httptest.NewRequest(http.MethodPost, "/sms/africastalking", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()

	handler.HandleAfricaTalking(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if sender.to != "+254712345678" {
		t.Fatalf("expected trimmed sender number, got %q", sender.to)
	}
	if !strings.Contains(sender.message, "Ongata Rongai Police") {
		t.Fatalf("unexpected SMS reply: %q", sender.message)
	}
}

func TestHandleAfricaTalkingRejectsMissingSender(t *testing.T) {
	handler := NewSMSHandler(service.NewSMSService(&smsHandlerSender{}, "kamilimu", "ignition"))
	req := httptest.NewRequest(http.MethodPost, "/sms/africastalking", strings.NewReader("text=help"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()

	handler.HandleAfricaTalking(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", recorder.Code)
	}
}
