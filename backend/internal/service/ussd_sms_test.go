package service

import (
	"strings"
	"testing"

	"motoguard-backend/internal/domain"
)

func TestCleanSMSInput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"trims and collapses spaces", "  police   rongai ", "police rongai"},
		{"menu separators are removed", "1*2#", "1 2"},
		{"empty stays empty", "   ", ""},
		{"zero is kept for back navigation", "0", "0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cleanSMSInput(tt.input); got != tt.expected {
				t.Errorf("cleanSMSInput(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestCleanSMSInputLimit(t *testing.T) {
	long := strings.Repeat("a", smsMaxInput+25)
	if got := cleanSMSInput(long); len([]rune(got)) != smsMaxInput {
		t.Errorf("expected input capped at %d runes, got %d", smsMaxInput, len([]rune(got)))
	}
}

func TestJoinSMSSteps(t *testing.T) {
	if got := joinSMSSteps("", "2"); got != "2" {
		t.Errorf("first reply: got %q, want %q", got, "2")
	}
	if got := joinSMSSteps("2", "1"); got != "2*1" {
		t.Errorf("second reply: got %q, want %q", got, "2*1")
	}
}

// A conversation built from SMS replies must parse to the same navigation stack as USSD.
func TestSMSStepsMatchUSSDStack(t *testing.T) {
	steps := ""
	for _, reply := range []string{"4", "2", "rongai"} {
		steps = joinSMSSteps(steps, reply)
	}
	stack := parseUSSDStack(steps)
	if len(stack) != 3 || stack[0] != "4" || stack[1] != "2" || stack[2] != "rongai" {
		t.Fatalf("unexpected stack for %q: %v", steps, stack)
	}

	steps = joinSMSSteps(joinSMSSteps("2", "0"), "1")
	if stack := parseUSSDStack(steps); len(stack) != 1 || stack[0] != "1" {
		t.Fatalf("back navigation not applied for %q: %v", steps, stack)
	}
}

func TestSplitSMSReply(t *testing.T) {
	body, open := splitSMSReply("CON Welcome to M-Guard\n1. Find bike")
	if !open || body != "Welcome to M-Guard\n1. Find bike" {
		t.Errorf("CON reply: got open=%v body=%q", open, body)
	}

	body, open = splitSMSReply("END Immobilization requested.")
	if open || body != "Immobilization requested." {
		t.Errorf("END reply: got open=%v body=%q", open, body)
	}
}

func TestSMSMenuMatchesUSSDMenu(t *testing.T) {
	s := &USSDService{}
	bike := testSMSBike()

	ussd := s.handleSingleBikeMenu(nil, "+254712345678", "", bike, nil)
	body, open := splitSMSReply(ussd)
	if !open || !strings.Contains(body, "1. Find bike") || !strings.Contains(body, "4. Emergency phones") {
		t.Fatalf("SMS welcome body does not match USSD menu: %q", body)
	}
}

func testSMSBike() domain.RiderBikeInfo {
	return domain.RiderBikeInfo{
		RegistrationNumber: "KMDF 123X",
		Status:             "STOPPED",
		BatteryPercent:     90,
		CurrentAddress:     "Haile Selassie Ave, Nairobi",
	}
}

func TestRouteSMSText(t *testing.T) {
	s := &USSDService{}
	s.SetSMSRouting("kamilimu")

	tests := []struct {
		name  string
		text  string
		rest  string
		forUs bool
	}{
		{"full prefix routes to us", "kamilimu mguard police Rongai", "police Rongai", true},
		{"prefix only opens the menu", "kamilimu mguard", "menu", true},
		{"prefix is case-insensitive", "KAMILIMU MGuard 2", "2", true},
		{"another project is ignored", "kamilimu ignition police Rongai", "", false},
		{"routing word without project is ignored", "kamilimu", "", false},
		{"gateway-stripped project prefix", "mguard 1", "1", true},
		{"bare text passes through", "2*1", "2*1", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rest, forUs := s.routeSMSText(tt.text)
			if rest != tt.rest || forUs != tt.forUs {
				t.Errorf("routeSMSText(%q) = (%q, %v), want (%q, %v)", tt.text, rest, forUs, tt.rest, tt.forUs)
			}
		})
	}
}
