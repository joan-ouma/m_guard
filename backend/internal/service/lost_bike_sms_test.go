package service

import (
	"strings"
	"testing"
)

func TestLostBikeMessageFormat(t *testing.T) {
	got := lostBikeMessage("en", "Jane Wanjiru", "KDA 123A", "Kimathi Street, Nairobi")
	want := "Jane Wanjiru has lost their bike KDA 123A. Last location is Kimathi Street, Nairobi."
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestLostBikeMessageFallbacks(t *testing.T) {
	got := lostBikeMessage("en", "  ", "KDA 123A", "")
	if !strings.HasPrefix(got, "A MotorGuard rider has lost their bike KDA 123A.") {
		t.Errorf("missing name fallback: %q", got)
	}
	if !strings.HasSuffix(got, "Last location is unknown.") {
		t.Errorf("missing address fallback: %q", got)
	}
}

func TestLostBikeMessageFitsOneSegment(t *testing.T) {
	long := strings.Repeat("Mombasa Road near the big roundabout ", 10)
	got := lostBikeMessage("en", "Jane Wanjiru", "KDA 123A", long)
	if len(got) > lostBikeSMSMaxLength {
		t.Errorf("message is %d characters, want at most %d: %q", len(got), lostBikeSMSMaxLength, got)
	}
	if !strings.HasSuffix(got, "...") {
		t.Errorf("expected truncated address to end with ...: %q", got)
	}
}

func TestLostBikeMessageKiswahili(t *testing.T) {
	got := lostBikeMessage("sw", "Jane Wanjiru", "KDA 123A", "Kimathi Street, Nairobi")
	want := "Jane Wanjiru amepoteza pikipiki yake KDA 123A. Mahali pa mwisho ni Kimathi Street, Nairobi."
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	fallback := lostBikeMessage("sw", "  ", "KDA 123A", "")
	if !strings.HasPrefix(fallback, "Mpanda pikipiki wa MotorGuard amepoteza pikipiki yake KDA 123A.") {
		t.Errorf("missing name fallback: %q", fallback)
	}
	if !strings.HasSuffix(fallback, "Mahali pa mwisho ni haijulikani.") {
		t.Errorf("missing address fallback: %q", fallback)
	}

	long := strings.Repeat("Mombasa Road near the big roundabout ", 10)
	if got := lostBikeMessage("sw", "Jane Wanjiru", "KDA 123A", long); len(got) > lostBikeSMSMaxLength {
		t.Errorf("message is %d characters, want at most %d: %q", len(got), lostBikeSMSMaxLength, got)
	}
}
