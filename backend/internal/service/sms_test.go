package service

import (
	"context"
	"testing"

	"motoguard-backend/internal/domain"
)

type testSMSReplySender struct {
	to      string
	message string
}

func (s *testSMSReplySender) SendSMS(_ context.Context, to, message string) error {
	s.to = to
	s.message = message
	return nil
}

func TestSMSServiceHandleIncoming(t *testing.T) {
	sender := &testSMSReplySender{}
	svc := NewSMSService(sender, "kamilimu", "ignition")

	err := svc.HandleIncoming(context.Background(), domain.SMSIncomingRequest{
		From: "+254712345678",
		Text: "kamilimu ignition police rongai",
	})
	if err != nil {
		t.Fatalf("HandleIncoming returned error: %v", err)
	}
	if sender.to != "+254712345678" {
		t.Fatalf("expected reply to sender, got %q", sender.to)
	}
	if !containsString(sender.message, "Ongata Rongai Police") || !containsString(sender.message, "0721 334 999") {
		t.Fatalf("unexpected reply: %q", sender.message)
	}
}

func TestSMSServiceIgnoresOtherRoutedProject(t *testing.T) {
	sender := &testSMSReplySender{}
	svc := NewSMSService(sender, "kamilimu", "ignition")

	if err := svc.HandleIncoming(context.Background(), domain.SMSIncomingRequest{
		From: "+254712345678",
		Text: "kamilimu another-project police rongai",
	}); err != nil {
		t.Fatalf("HandleIncoming returned error: %v", err)
	}
	if sender.message != "" {
		t.Fatalf("expected no reply for another project, got %q", sender.message)
	}
}

func TestSMSServiceRequestsAreaForPoliceLookup(t *testing.T) {
	sender := &testSMSReplySender{}
	svc := NewSMSService(sender, "kamilimu", "ignition")

	if err := svc.HandleIncoming(context.Background(), domain.SMSIncomingRequest{
		From: "+254712345678",
		Text: "police",
	}); err != nil {
		t.Fatalf("HandleIncoming returned error: %v", err)
	}
	if !containsString(sender.message, "POLICE Rongai") {
		t.Fatalf("expected area guidance, got %q", sender.message)
	}
}
