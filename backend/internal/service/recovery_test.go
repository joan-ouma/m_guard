package service

import (
	"context"
	"testing"
	"time"

	"motoguard-backend/internal/domain"
)

// These tests cover validation only. They return before any repository call, so no database is needed.

func badRequestErrors(t *testing.T, err error) map[string][]string {
	t.Helper()
	pd, ok := err.(domain.ProblemDetails)
	if !ok {
		t.Fatalf("expected ProblemDetails, got %T: %v", err, err)
	}
	if pd.Status != 400 {
		t.Fatalf("expected 400, got %d", pd.Status)
	}
	return pd.Errors
}

func TestCreateIncidentValidation(t *testing.T) {
	s := NewRecoveryService(nil)
	valid := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)

	cases := []struct {
		name  string
		req   domain.CreateIncidentRequest
		field string
	}{
		{"unknown mode", domain.CreateIncidentRequest{Mode: "stolen_by_ghost", LostAt: valid, LocationLost: "Kimathi St"}, "mode"},
		{"bad time", domain.CreateIncidentRequest{Mode: "other", LostAt: "yesterday", LocationLost: "Kimathi St"}, "lost_at"},
		{"future time", domain.CreateIncidentRequest{Mode: "other", LostAt: time.Now().Add(time.Hour).UTC().Format(time.RFC3339), LocationLost: "Kimathi St"}, "lost_at"},
		{"empty location", domain.CreateIncidentRequest{Mode: "other", LostAt: valid, LocationLost: "  "}, "location_lost"},
		{"lat without lng", domain.CreateIncidentRequest{Mode: "other", LostAt: valid, LocationLost: "x", Latitude: ptrFloat(1)}, "latitude"},
		{"lat out of range", domain.CreateIncidentRequest{Mode: "other", LostAt: valid, LocationLost: "x", Latitude: ptrFloat(95), Longitude: ptrFloat(10)}, "latitude"},
		{"bad bike id", domain.CreateIncidentRequest{Mode: "other", LostAt: valid, LocationLost: "x", BikeID: "not-a-uuid"}, "bike_id"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := s.CreateIncident(context.Background(), "user-1", tc.req)
			errs := badRequestErrors(t, err)
			if _, ok := errs[tc.field]; !ok {
				t.Errorf("expected error on %q, got %v", tc.field, errs)
			}
		})
	}
}

func TestCreateContactValidation(t *testing.T) {
	s := NewRecoveryService(nil)

	_, err := s.CreateContact(context.Background(), "user-1", domain.CreateContactRequest{
		Name: "", Phone: "12", Category: "friend",
	})
	errs := badRequestErrors(t, err)
	for _, field := range []string{"name", "phone", "category"} {
		if _, ok := errs[field]; !ok {
			t.Errorf("expected error on %q, got %v", field, errs)
		}
	}
}

func TestSetOBValidation(t *testing.T) {
	s := NewRecoveryService(nil)
	id := "3f6c2a1e-9b4d-4e8a-8f2a-1c2d3e4f5a6b"

	if err := s.SetOB(context.Background(), "user-1", id, "x", nil); err == nil {
		t.Error("expected error for too-short OB number")
	} else {
		badRequestErrors(t, err)
	}

	if err := s.SetOB(context.Background(), "user-1", "not-a-uuid", "OB 12/24/2026", nil); err == nil {
		t.Error("expected not found for invalid incident id")
	}

	big := &domain.IncidentAttachment{FileName: "big.pdf", Data: make([]byte, MaxAttachmentBytes+1)}
	if err := s.SetOB(context.Background(), "user-1", id, "OB 12/24/2026", big); err == nil {
		t.Error("expected error for oversized attachment")
	} else {
		badRequestErrors(t, err)
	}

	// Plain text pretending to be a PDF must be rejected by content sniffing.
	fake := &domain.IncidentAttachment{FileName: "receipt.pdf", Data: []byte("just some text, not a pdf")}
	if err := s.SetOB(context.Background(), "user-1", id, "OB 12/24/2026", fake); err == nil {
		t.Error("expected error for disallowed content")
	} else {
		badRequestErrors(t, err)
	}
}

func TestSanitizeFileName(t *testing.T) {
	cases := map[string]string{
		"receipt.pdf":           "receipt.pdf",
		"../../etc/passwd":      "passwd",
		`C:\Users\rider\ob.png`: "ob.png",
		"":                      "receipt",
	}
	for in, want := range cases {
		if got := sanitizeFileName(in); got != want {
			t.Errorf("sanitizeFileName(%q) = %q, want %q", in, got, want)
		}
	}
}

func ptrFloat(f float64) *float64 { return &f }
