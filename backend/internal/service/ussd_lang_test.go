package service

import (
	"strings"
	"testing"
)

func TestSMSLanguageCommand(t *testing.T) {
	tests := []struct {
		input string
		lang  string
		ok    bool
	}{
		{"SW", langSW, true},
		{" kiswahili ", langSW, true},
		{"Swahili", langSW, true},
		{"en", langEN, true},
		{"ENGLISH", langEN, true},
		{"1", "", false},
		{"menu", "", false},
	}
	for _, tt := range tests {
		lang, ok := smsLanguageCommand(tt.input)
		if lang != tt.lang || ok != tt.ok {
			t.Errorf("smsLanguageCommand(%q) = (%q, %v), want (%q, %v)", tt.input, lang, ok, tt.lang, tt.ok)
		}
	}
}

func TestNormalizeLangFallsBackToEnglish(t *testing.T) {
	if normalizeLang("sw") != langSW || normalizeLang("en") != langEN {
		t.Error("supported languages should pass through")
	}
	if normalizeLang("") != langEN || normalizeLang("fr") != langEN {
		t.Error("unsupported or empty language should fall back to English")
	}
}

func TestSingleBikeWelcomeInKiswahili(t *testing.T) {
	s := &USSDService{}
	body, open := splitSMSReply(s.handleSingleBikeMenu(nil, "+254712345678", "", langSW, testSMSBike(), nil))
	if !open || !strings.Contains(body, "Karibu M-Guard") || !strings.Contains(body, "1. Tafuta pikipiki") {
		t.Fatalf("expected Kiswahili welcome menu, got open=%v body=%q", open, body)
	}
}

func TestStatusAndEmergencyInKiswahili(t *testing.T) {
	s := &USSDService{}
	bike := testSMSBike()
	bike.Status = "" // no stored status, so the Kiswahili fallback label is used
	status := s.renderBikeStatus(bike, langSW)
	if !strings.Contains(status, "Hali: IMESIMAMA") || !strings.Contains(status, "Mahali: Haile Selassie Ave, Nairobi") {
		t.Errorf("status not in Kiswahili: %q", status)
	}

	search := searchPoliceStations("zzzz", langSW)
	if !strings.HasPrefix(search, "CON Hakuna kituo kilichopatikana") {
		t.Errorf("no-result search not in Kiswahili: %q", search)
	}
}

func TestEnglishMenuUnchangedByLanguageSupport(t *testing.T) {
	s := &USSDService{}
	body, _ := splitSMSReply(s.handleSingleBikeMenu(nil, "+254712345678", "", langEN, testSMSBike(), nil))
	if !strings.Contains(body, "Welcome to M-Guard") || !strings.Contains(body, "4. Emergency phones") {
		t.Errorf("English menu changed: %q", body)
	}
	if got := searchPoliceStations("zzzz", langEN); !strings.Contains(got, "No station found for") {
		t.Errorf("English search text changed: %q", got)
	}
}

func TestLanguageChangeConfirmationIsInChosenLanguage(t *testing.T) {
	if !strings.Contains(languageChangedText(langSW), "Kiswahili") {
		t.Error("Kiswahili confirmation should be written in Kiswahili")
	}
	if !strings.Contains(languageChangedText(langEN), "English") {
		t.Error("English confirmation should be written in English")
	}
}
