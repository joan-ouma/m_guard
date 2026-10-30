package service

import (
	"testing"

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

func TestParseUSSDStack(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"", nil},
		{"1", []string{"1"}},
		{"1*0", nil},             // Pressed 1, then went back
		{"1*0*2", []string{"2"}}, // Pressed 1, went back, then pressed 2
		{"2*1", []string{"2", "1"}},
		{"4*0", nil}, // Pressed emergency phones, then went back
	}

	for _, tt := range tests {
		actual := parseUSSDStack(tt.input)
		if len(actual) != len(tt.expected) {
			t.Fatalf("for input %q: expected len %d, got %d (%v)", tt.input, len(tt.expected), len(actual), actual)
		}
		for i := range actual {
			if actual[i] != tt.expected[i] {
				t.Errorf("for input %q at idx %d: expected %s, got %s", tt.input, i, tt.expected[i], actual[i])
			}
		}
	}
}

func TestUSSDSingleBikeMenuFlow(t *testing.T) {
	s := &USSDService{}
	bike := domain.RiderBikeInfo{
		RegistrationNumber: "KMDF 123X",
		Status:             "STOPPED",
		Ignition:           false,
		SpeedKMH:           0,
		BatteryPercent:     90,
		CurrentAddress:     "Haile Selassie Ave, Nairobi",
		IsImmobilized:      false,
	}

	// 1. Root Welcome menu
	welcome := s.handleSingleBikeMenu(nil, "+254712345678", "USSD (*384#)", bike, nil)
	if !containsString(welcome, "CON Welcome to M-Guard") ||
		!containsString(welcome, "1. Find bike") ||
		!containsString(welcome, "2. Immobilize bike") ||
		!containsString(welcome, "3. Reactivate bike") ||
		!containsString(welcome, "4. Emergency phones") {
		t.Fatalf("unexpected welcome menu: %s", welcome)
	}

	// 2. Option 1: Find bike
	findBike := s.handleSingleBikeMenu(nil, "+254712345678", "USSD (*384#)", bike, []string{"1"})
	if !containsString(findBike, "CON Bike KMDF 123X") || !containsString(findBike, "0. Back") {
		t.Fatalf("unexpected find bike response: %s", findBike)
	}

	// 3. Option 2: Immobilize bike confirmation
	immob := s.handleSingleBikeMenu(nil, "+254712345678", "USSD (*384#)", bike, []string{"2"})
	if !containsString(immob, "CON Turn off engine for KMDF 123X?") || !containsString(immob, "0. Back") {
		t.Fatalf("unexpected immobilize confirmation: %s", immob)
	}

	// 4. Option 3: Reactivate bike (when bike is immobilized)
	bikeImmob := bike
	bikeImmob.IsImmobilized = true
	react := s.handleSingleBikeMenu(nil, "+254712345678", "USSD (*384#)", bikeImmob, []string{"3"})
	if !containsString(react, "CON Restore engine for KMDF 123X?") || !containsString(react, "0. Back") {
		t.Fatalf("unexpected reactivate confirmation: %s", react)
	}

	// 5. Option 4: Emergency directory & search
	emergBike := bike
	emergBike.OwnerName = "Alpha Fleet"
	emergBike.OwnerPhone = "+254700000000"
	emergMenu := s.handleSingleBikeMenu(nil, "+254712345678", "USSD (*384#)", emergBike, []string{"4"})
	if !containsString(emergMenu, "CON Emergency Directory:") || !containsString(emergMenu, "1. Fleet Owner") || !containsString(emergMenu, "2. Search Police Station") {
		t.Fatalf("unexpected emergency directory menu: %s", emergMenu)
	}

	emergOwner := s.handleSingleBikeMenu(nil, "+254712345678", "USSD (*384#)", emergBike, []string{"4", "1"})
	if !containsString(emergOwner, "CON Fleet Owner:") || !containsString(emergOwner, "Alpha Fleet: +254700000000") {
		t.Fatalf("unexpected fleet owner response: %s", emergOwner)
	}

	emergSearchPrompt := s.handleSingleBikeMenu(nil, "+254712345678", "USSD (*384#)", emergBike, []string{"4", "2"})
	if !containsString(emergSearchPrompt, "CON Enter town or station name") {
		t.Fatalf("unexpected search prompt: %s", emergSearchPrompt)
	}

	emergSearchRes := s.handleSingleBikeMenu(nil, "+254712345678", "USSD (*384#)", emergBike, []string{"4", "2", "rongai"})
	if !containsString(emergSearchRes, "Ongata Rongai Police") || !containsString(emergSearchRes, "0721 334 999") {
		t.Fatalf("unexpected search results: %s", emergSearchRes)
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
