package geolocation

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCellResolver_Resolve(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			// OpenCelliD mock
			_ = json.NewEncoder(w).Encode(map[string]float64{
				"lat": -1.286389,
				"lon": 36.817223,
			})
			return
		}
		// BeaconDB mock
		_ = json.NewEncoder(w).Encode(map[string]any{
			"location": map[string]float64{
				"lat": -1.286389,
				"lng": 36.817223,
			},
			"accuracy": 150.0,
		})
	}))
	defer ts.Close()

	resolver := NewCellResolver()
	resolver.endpoint = ts.URL

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	lat, lng, ok := resolver.Resolve(ctx, 639, 2, 2358, 9940)
	if !ok {
		t.Fatal("expected cell resolution to succeed")
	}
	if lat != -1.286389 || lng != 36.817223 {
		t.Fatalf("unexpected coordinates: got (%f, %f)", lat, lng)
	}

	// Verify caching - shut down server, subsequent resolve should still hit cache
	ts.Close()
	latCached, lngCached, okCached := resolver.Resolve(ctx, 639, 2, 2358, 9940)
	if !okCached || latCached != lat || lngCached != lng {
		t.Fatalf("expected cached coordinates: got (%f, %f, %v)", latCached, lngCached, okCached)
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
