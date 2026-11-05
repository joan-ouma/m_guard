package config

import (
	"os"
	"testing"
)

func TestConfigLoad_Mailjet(t *testing.T) {
	os.Setenv("DATABASE_URL", "postgres://test:test@localhost:5432/test")
	os.Setenv("JWT_SECRET", "super-secret-jwt-key-minimum-32bytes")
	os.Setenv("MAILJET_API_KEY", "test-api-key")
	os.Setenv("MAILJET_SECRET_KEY", "test-secret-key")
	os.Setenv("MAILJET_SENDER_EMAIL", "no-reply@jullie.tech")
	os.Setenv("MAILJET_REPLY_TO_EMAIL", "hello@jullie.tech")
	os.Setenv("MAILJET_SENDER_NAME", "MotoGuard")
	os.Setenv("FRONTEND_URL", "http://localhost:4200")
	os.Setenv("LOG_FILE", "/var/log/motoguard.log")
	os.Setenv("AT_USERNAME", "buildathon-voice")
	os.Setenv("AT_API_KEY", "test-at-key")
	os.Setenv("AT_SMS_FROM", "20384")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.LogFile != "/var/log/motoguard.log" {
		t.Errorf("expected LogFile %q, got %q", "/var/log/motoguard.log", cfg.LogFile)
	}

	if cfg.MailjetAPIKey != "test-api-key" {
		t.Errorf("expected mailjet api key %q, got %q", "test-api-key", cfg.MailjetAPIKey)
	}
	if cfg.MailjetSecretKey != "test-secret-key" {
		t.Errorf("expected mailjet secret key %q, got %q", "test-secret-key", cfg.MailjetSecretKey)
	}
	if cfg.MailjetSenderEmail != "no-reply@jullie.tech" {
		t.Errorf("expected sender email %q, got %q", "no-reply@jullie.tech", cfg.MailjetSenderEmail)
	}
	if cfg.MailjetReplyToEmail != "hello@jullie.tech" {
		t.Errorf("expected reply-to email %q, got %q", "hello@jullie.tech", cfg.MailjetReplyToEmail)
	}
	if cfg.MailjetSenderName != "MotoGuard" {
		t.Errorf("expected sender name %q, got %q", "MotoGuard", cfg.MailjetSenderName)
	}
	if cfg.FrontendURL != "http://localhost:4200" {
		t.Errorf("expected frontend URL %q, got %q", "http://localhost:4200", cfg.FrontendURL)
	}
	if cfg.MQTTBrokerURL != "tcp://localhost:1883" {
		t.Errorf("expected default MQTTBrokerURL, got %q", cfg.MQTTBrokerURL)
	}
	if cfg.MQTTUsername != "tracker" {
		t.Errorf("expected default MQTTUsername tracker, got %q", cfg.MQTTUsername)
	}
	if cfg.MQTTPassword != "~:!Nu2>#6TDe5Y+" {
		t.Errorf("expected default MQTTPassword, got %q", cfg.MQTTPassword)
	}
	if cfg.ATUsername != "buildathon-voice" || cfg.ATAPIKey != "test-at-key" || cfg.ATSMSFrom != "20384" {
		t.Errorf("unexpected Africa's Talking configuration: username=%q from=%q", cfg.ATUsername, cfg.ATSMSFrom)
	}
	if cfg.SMSRoutingKeyword != "kamilimu" || cfg.SMSProjectKeyword != "ignition" {
		t.Errorf("unexpected SMS routing defaults: routing=%q project=%q", cfg.SMSRoutingKeyword, cfg.SMSProjectKeyword)
	}
}
