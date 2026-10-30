package domain

import (
	"encoding/json"
	"time"
)

type Geofence struct {
	ID             string
	OrganizationID string
	CreatedBy      string
	Name           string
	Description    string
	CenterLat      float64
	CenterLng      float64
	RadiusMeters   float64
	AlertOnExit    bool
	AlertOnEntry   bool
	Active         bool
	Status         string
	SuspendedAt    *time.Time
	DeletedAt      *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type BikeBreadcrumb struct {
	ID             int64
	BikeID         string
	OrganizationID string
	Latitude       float64
	Longitude      float64
	HeadingDeg     float64
	SpeedKMH       float64
	Ignition       bool
	BatteryPercent int
	Address        string
	GeofenceBreach bool
	RecordedAt     time.Time
}

type Tracker struct {
	ID              string
	IMEI            string
	PhoneNumber     string
	SIMICCID        string
	Model           string
	HardwareVersion string
	SoftwareVersion string
	Status          string
	LastOnlineAt    *time.Time
	Notes           string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type TrackerAssignment struct {
	ID             string
	TrackerID      string
	BikeID         string
	AssignedBy     string
	AssignedAt     time.Time
	UnassignedBy   string
	UnassignedAt   *time.Time
	UnassignReason string
	Status         string
	Notes          string
	CreatedAt      time.Time
}

type CellTower struct {
	LAC     int `json:"lac"`
	CID     int `json:"cid"`
	RSSIDBm int `json:"dbm"`
}

type TelemetryPayload struct {
	SensorID       string      `json:"sensor_id"`
	Latitude       float64     `json:"lat"`
	Longitude      float64     `json:"lng"`
	LocType        string      `json:"loc_type,omitempty"`
	Accuracy       float64     `json:"accuracy,omitempty"`
	MCC            int         `json:"mcc,omitempty"`
	MNC            int         `json:"mnc,omitempty"`
	LAC            int         `json:"lac,omitempty"`
	CID            int         `json:"cid,omitempty"`
	Cells          []CellTower `json:"cells,omitempty"`
	SignalQuality  int         `json:"csq,omitempty"`
	RSSIDBm        int         `json:"rssi_dbm,omitempty"`
	SpeedKMH       float64     `json:"speed"`
	HeadingDeg     float64     `json:"heading"`
	Ignition       bool        `json:"ign"`
	BatteryPercent int         `json:"bat,omitempty"`
	Timestamp      int64       `json:"ts,omitempty"`
	Alert          bool        `json:"alert,omitempty"`
	Reason         string      `json:"reason,omitempty"`
	Event          string      `json:"event,omitempty"`
	EventType      string      `json:"event_type,omitempty"`
}

func (p *TelemetryPayload) UnmarshalJSON(data []byte) error {
	type Alias TelemetryPayload
	aux := &struct {
		Spd   *float64 `json:"spd"`
		Hdg   *float64 `json:"hdg"`
		Event *string  `json:"event"`
		*Alias
	}{
		Alias: (*Alias)(p),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if aux.Spd != nil && p.SpeedKMH == 0 {
		p.SpeedKMH = *aux.Spd
	}
	if aux.Hdg != nil && p.HeadingDeg == 0 {
		p.HeadingDeg = *aux.Hdg
	}
	if p.EventType == "" && aux.Event != nil {
		p.EventType = *aux.Event
	}
	if p.EventType == "" && p.Reason != "" {
		p.EventType = p.Reason
	}
	return nil
}

type AlertPayload struct {
	Alert          bool        `json:"alert"`
	Reason         string      `json:"reason"`
	Latitude       float64     `json:"lat"`
	Longitude      float64     `json:"lng"`
	LocType        string      `json:"loc_type,omitempty"`
	Accuracy       float64     `json:"accuracy,omitempty"`
	MCC            int         `json:"mcc,omitempty"`
	MNC            int         `json:"mnc,omitempty"`
	LAC            int         `json:"lac,omitempty"`
	CID            int         `json:"cid,omitempty"`
	Cells          []CellTower `json:"cells,omitempty"`
	SignalQuality  int         `json:"csq,omitempty"`
	RSSIDBm        int         `json:"rssi_dbm,omitempty"`
	SpeedKMH       float64     `json:"speed"`
	HeadingDeg     float64     `json:"heading"`
	Ignition       bool        `json:"ign"`
	Timestamp      int64       `json:"ts,omitempty"`
	BatteryPercent int         `json:"bat,omitempty"`
}

func (a *AlertPayload) UnmarshalJSON(data []byte) error {
	type Alias AlertPayload
	aux := &struct {
		Spd *float64 `json:"spd"`
		Hdg *float64 `json:"hdg"`
		*Alias
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if aux.Spd != nil && a.SpeedKMH == 0 {
		a.SpeedKMH = *aux.Spd
	}
	if aux.Hdg != nil && a.HeadingDeg == 0 {
		a.HeadingDeg = *aux.Hdg
	}
	return nil
}

type BreadcrumbDTO struct {
	ID             int64     `json:"id"`
	BikeID         string    `json:"bike_id"`
	Latitude       float64   `json:"latitude"`
	Longitude      float64   `json:"longitude"`
	Accuracy       float64   `json:"accuracy"`
	LocType        string    `json:"loc_type"`
	SpeedKMH       float64   `json:"speed_kmh"`
	HeadingDeg     float64   `json:"heading_deg"`
	Ignition       bool      `json:"ignition"`
	BatteryPercent int       `json:"battery_percent"`
	Address        string    `json:"address"`
	GeofenceBreach bool      `json:"geofence_breach"`
	RecordedAt     time.Time `json:"recorded_at"`
}

type BikeEventDTO struct {
	ID                 int64     `json:"id"`
	BikeID             string    `json:"bike_id"`
	RegistrationNumber string    `json:"registration_number,omitempty"`
	Model              string    `json:"model,omitempty"`
	EventType          string    `json:"event_type"`
	Latitude           float64   `json:"latitude"`
	Longitude          float64   `json:"longitude"`
	RecordedAt         time.Time `json:"recorded_at"`
}
