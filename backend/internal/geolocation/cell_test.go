package geolocation

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"motoguard-backend/internal/domain"
)

func TestCellResolver_Resolve(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"lat":   -1.286389,
			"lon":   36.817223,
			"range": 1500.0,
		})
	}))
	defer ts.Close()

	resolver := NewCellResolver()
	resolver.openCellIDEndpoint = ts.URL

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	lat, lng, ok := resolver.Resolve(ctx, 639, 2, 2358, 9940)
	if !ok {
		t.Fatal("expected cell resolution to succeed")
	}
	if lat != -1.286389 || lng != 36.817223 {
		t.Fatalf("unexpected coordinates: got (%f, %f)", lat, lng)
	}

	ts.Close()
	latCached, lngCached, okCached := resolver.Resolve(ctx, 639, 2, 2358, 9940)
	if !okCached || latCached != lat || lngCached != lng {
		t.Fatalf("expected cached coordinates: got (%f, %f, %v)", latCached, lngCached, okCached)
	}
}

func TestCellResolver_MultiCellMultilateration(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.RawQuery, "cell_id=1001") {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"lat":   -1.413000,
				"lon":   36.750000,
				"range": 1000.0,
			})
			return
		}
		if strings.Contains(r.URL.RawQuery, "cell_id=1002") {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"lat":   -1.415000,
				"lon":   36.752000,
				"range": 1200.0,
			})
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("false"))
	}))
	defer ts.Close()

	resolver := NewCellResolver()
	resolver.openCellIDEndpoint = ts.URL

	ctx := context.Background()
	cells := []domain.CellTower{
		{LAC: 2358, CID: 1001, RSSIDBm: -65},
		{LAC: 2358, CID: 1002, RSSIDBm: -75},
	}

	lat, lng, acc, ok := resolver.ResolveMulti(ctx, 639, 2, cells)
	if !ok {
		t.Fatal("expected multi-cell resolution to succeed")
	}
	if lat >= -1.413000 || lat <= -1.415000 {
		t.Fatalf("expected multilaterated lat between -1.415000 and -1.413000, got %f", lat)
	}
	if lng <= 36.750000 || lng >= 36.752000 {
		t.Fatalf("expected multilaterated lng between 36.750000 and 36.752000, got %f", lng)
	}
	if acc <= 0 {
		t.Fatalf("expected valid positive accuracy, got %f", acc)
	}
}

func TestCellResolver_NotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("false"))
	}))
	defer ts.Close()

	resolver := NewCellResolver()
	resolver.openCellIDEndpoint = ts.URL

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, _, ok := resolver.Resolve(ctx, 639, 2, 9999, 9999)
	if ok {
		t.Fatal("expected resolution to fail for unknown cell")
	}
}

func TestCellResolver_FallbackToLastKnownCell(t *testing.T) {
	callCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		callCount++
		if callCount == 1 {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"lat":   -1.413026,
				"lon":   36.750668,
				"range": 1200.0,
			})
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("false"))
	}))
	defer ts.Close()

	resolver := NewCellResolver()
	resolver.openCellIDEndpoint = ts.URL

	ctx := context.Background()

	lat, lng, ok := resolver.Resolve(ctx, 639, 2, 2358, 1001)
	if !ok || lat != -1.413026 || lng != 36.750668 {
		t.Fatalf("expected initial resolution to succeed, got (%f, %f, %v)", lat, lng, ok)
	}

	lat2, lng2, ok2 := resolver.Resolve(ctx, 639, 2, 2358, 9999)
	if !ok2 || lat2 != -1.413026 || lng2 != 36.750668 {
		t.Fatalf("expected fallback to last known cell, got (%f, %f, %v)", lat2, lng2, ok2)
	}

	lat3, lng3, ok3 := resolver.ResolveMulti(ctx, 639, 2, []domain.CellTower{
		{LAC: 8888, CID: 7777, RSSIDBm: -80},
	})
	if !ok3 || lat3 != -1.413026 || lng3 != 36.750668 {
		t.Fatalf("expected fallback to last correct cell found, got (%f, %f, %v)", lat3, lng3, ok3)
	}
}

func TestCellResolver_InvalidParameters(t *testing.T) {
	resolver := NewCellResolver()
	ctx := context.Background()

	_, _, ok := resolver.Resolve(ctx, 0, 0, 0, 0)
	if ok {
		t.Fatal("expected failure on 0 values")
	}
}
