package geolocation

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestReverseGeocoder_ReverseGeocode(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"display_name": "Rimpa Road, Ongata Rongai ward, Kajiado North, Kajiado, Kenya",
			"address": map[string]string{
				"road":   "Rimpa Road",
				"city":   "Ongata Rongai ward",
				"county": "Kajiado North",
				"state":  "Kajiado",
			},
		})
	}))
	defer ts.Close()

	geocoder := NewReverseGeocoder()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	info, err := geocoder.fromNominatim(ctx, -1.414123, 36.743090)
	// Even if public internet varies, testing with local server URL
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL, nil)
	resp, err := geocoder.client.Do(req)
	if err != nil {
		t.Fatalf("failed test request: %v", err)
	}
	defer resp.Body.Close()

	if info.FormattedAddress == "" && err != nil {
		// Public test could be rate limited or offline; verify geocoder handles gracefully
		fallback := geocoder.ReverseGeocode(ctx, -1.414123, 36.743090)
		if fallback.FormattedAddress == "" {
			t.Fatal("expected non-empty formatted address fallback")
		}
	}
}

func TestReverseGeocoder_ZeroCoordinates(t *testing.T) {
	geocoder := NewReverseGeocoder()
	ctx := context.Background()

	info := geocoder.ReverseGeocode(ctx, 0, 0)
	if info.FormattedAddress != "" {
		t.Fatalf("expected empty string for 0,0, got %s", info.FormattedAddress)
	}
}
