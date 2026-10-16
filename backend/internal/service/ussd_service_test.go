package service

import (
	"testing"

	"motoguard-backend/internal/africastalking"
	"motoguard-backend/internal/domain"
)

func TestHashPIN(t *testing.T) {
	phone := "+254712345678"
	pin := "1234"

	hash1 := HashPIN(phone, pin)
	hash2 := HashPIN(phone, pin)

	if hash1 == "" {
		t.Fatal("expected non-empty pin hash")
	}
	if hash1 != hash2 {
		t.Fatal("expected deterministic pin hash for same phone & pin")
	}

	diffHash := HashPIN(phone, "5678")
	if hash1 == diffHash {
		t.Fatal("expected different hash for different pin")
	}
}

func TestRenderBikeStatus(t *testing.T) {
	s := &USSDService{}
	bike := domain.RiderBikeInfo{
		RegistrationNumber: "KME 123A",
		Make:               "Boxer",
		Status:             "RUNNING",
		Ignition:           true,
		SpeedKMH:           38.5,
		BatteryPercent:     88,
		IsImmobilized:      false,
		CurrentAddress:     "Magadi Road, Ongata Rongai",
	}

	statusStr := s.renderBikeStatus(bike)
	if !containsString(statusStr, "KME 123A") {
		t.Errorf("expected registration in status string: %s", statusStr)
	}
	if !containsString(statusStr, "RUNNING") {
		t.Errorf("expected status RUNNING in status string: %s", statusStr)
	}
	if !containsString(statusStr, "Magadi Road") {
		t.Errorf("expected address in status string: %s", statusStr)
	}
}

func TestIncomingSMSCommandParsing(t *testing.T) {
	atClient := africastalking.NewClient("sandbox", "", true)
	s := NewUSSDService(nil, atClient, nil, nil, nil)

	bikes := []domain.RiderBikeInfo{
		{
			RegistrationNumber: "KME 123A",
			Status:             "STOPPED",
			Ignition:           false,
			SpeedKMH:           0,
			BatteryPercent:     95,
			CurrentAddress:     "Westlands, Nairobi",
			IsImmobilized:      false,
		},
	}

	// Test STATUS command
	statusReply := s.handleSMSStatus(bikes, nil)
	if !containsString(statusReply, "KME 123A") || !containsString(statusReply, "STOPPED") {
		t.Errorf("unexpected status reply: %s", statusReply)
	}

	// Test Help command
	helpReply := s.smsHelpMessage()
	if !containsString(helpReply, "STATUS") || !containsString(helpReply, "LOCK") {
		t.Errorf("unexpected help reply: %s", helpReply)
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > len(substr) && stringIndex(s, substr) >= 0))
}

func stringIndex(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
