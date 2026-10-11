package domain

import (
	"encoding/json"
	"testing"
)

func TestTelemetryPayload_FirmwareUnmarshaling(t *testing.T) {
	raw := `{
		"lat": -1.286389,
		"lng": 36.817223,
		"loc_type": "gps",
		"mcc": 639,
		"mnc": 2,
		"lac": 2358,
		"cid": 9940,
		"csq": 22,
		"rssi_dbm": -69,
		"speed": 42.5,
		"heading": 182.4,
		"ign": true
	}`

	var p TelemetryPayload
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		t.Fatalf("failed to unmarshal firmware telemetry: %v", err)
	}

	if p.Latitude != -1.286389 || p.Longitude != 36.817223 {
		t.Errorf("unexpected coordinates: (%f, %f)", p.Latitude, p.Longitude)
	}
	if p.LocType != "gps" {
		t.Errorf("expected loc_type gps, got %s", p.LocType)
	}
	if p.MCC != 639 || p.MNC != 2 || p.LAC != 2358 || p.CID != 9940 {
		t.Errorf("unexpected cell info: mcc=%d, mnc=%d, lac=%d, cid=%d", p.MCC, p.MNC, p.LAC, p.CID)
	}
	if p.SpeedKMH != 42.5 {
		t.Errorf("expected speed 42.5, got %f", p.SpeedKMH)
	}
	if p.HeadingDeg != 182.4 {
		t.Errorf("expected heading 182.4, got %f", p.HeadingDeg)
	}
	if !p.Ignition {
		t.Errorf("expected ignition true, got %v", p.Ignition)
	}
}

func TestAlertPayload_FirmwareUnmarshaling(t *testing.T) {
	raw := `{
		"alert": true,
		"reason": "unauthorized_motion",
		"lat": -1.286389,
		"lng": 36.817223,
		"loc_type": "gps",
		"csq": 22,
		"rssi_dbm": -69,
		"spd": 5.4,
		"hdg": 182.4,
		"ign": false
	}`

	var a AlertPayload
	if err := json.Unmarshal([]byte(raw), &a); err != nil {
		t.Fatalf("failed to unmarshal firmware alert: %v", err)
	}

	if !a.Alert {
		t.Errorf("expected alert true")
	}
	if a.Reason != "unauthorized_motion" {
		t.Errorf("expected reason unauthorized_motion, got %s", a.Reason)
	}
	if a.SpeedKMH != 5.4 {
		t.Errorf("expected speed 5.4, got %f", a.SpeedKMH)
	}
	if a.HeadingDeg != 182.4 {
		t.Errorf("expected heading 182.4, got %f", a.HeadingDeg)
	}
	if a.Ignition {
		t.Errorf("expected ignition false")
	}
}
