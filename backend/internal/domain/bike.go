package domain

import (
	"time"
)

type Bike struct {
	ID                 string
	UserID             string
	RegistrationNumber string
	Label              string
	Make               string
	Model              string
	Year               int
	Color              string
	IMEI               string
	Status             string
	Ignition           bool
	SpeedKMH           float64
	HeadingDeg         float64
	Latitude           float64
	Longitude          float64
	CurrentAddress     string
	SubCounty          string
	County             string
	BatteryPercent     int
	SignalQualityCSQ   int
	LastTelemetryAt    *time.Time
	GeofenceID         string
	DriverContactID    string
	IsImmobilized      bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type BikeDTO struct {
	ID                 string     `json:"id"`
	UserID             string     `json:"user_id"`
	RegistrationNumber string     `json:"registration_number"`
	Label              string     `json:"label"`
	Make               string     `json:"make"`
	Model              string     `json:"model"`
	Year               int        `json:"year"`
	Color              string     `json:"color"`
	IMEI               string     `json:"imei"`
	Status             string     `json:"status"`
	Ignition           bool       `json:"ignition"`
	SpeedKMH           float64    `json:"speed_kmh"`
	HeadingDeg         float64    `json:"heading_deg"`
	Latitude           float64    `json:"latitude"`
	Longitude          float64    `json:"longitude"`
	CurrentAddress     string     `json:"current_address"`
	SubCounty          string     `json:"sub_county"`
	County             string     `json:"county"`
	BatteryPercent     int        `json:"battery_percent"`
	SignalQualityCSQ   int        `json:"signal_quality_csq"`
	LastTelemetryAt    *time.Time `json:"last_telemetry_at,omitempty"`
	GeofenceID         string     `json:"geofence_id,omitempty"`
	DriverContactID    string     `json:"driver_contact_id,omitempty"`
	IsImmobilized      bool       `json:"is_immobilized"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type ImmobilizeBikeRequest struct {
	Action string `json:"action"`
}

type CreateBikeRequest struct {
	RegistrationNumber string  `json:"registration_number"`
	Label              string  `json:"label"`
	Make               string  `json:"make"`
	Model              string  `json:"model"`
	Year               int     `json:"year"`
	Color              string  `json:"color"`
	IMEI               string  `json:"imei"`
	Latitude           float64 `json:"latitude"`
	Longitude          float64 `json:"longitude"`
	CurrentAddress     string  `json:"current_address"`
	SubCounty          string  `json:"sub_county"`
	County             string  `json:"county"`
	GeofenceID         string  `json:"geofence_id"`
	DriverContactID    string  `json:"driver_contact_id"`
}

type UpdateBikeRequest struct {
	RegistrationNumber string `json:"registration_number"`
	Label              string `json:"label"`
	Make               string `json:"make"`
	Model              string `json:"model"`
	Year               int    `json:"year"`
	Color              string `json:"color"`
	CurrentAddress     string `json:"current_address"`
	SubCounty          string `json:"sub_county"`
	County             string `json:"county"`
	Status             string `json:"status"`
	GeofenceID         string `json:"geofence_id"`
	DriverContactID    string `json:"driver_contact_id"`
}

type BikeListMeta struct {
	Total                 int    `json:"total"`
	Limit                 int    `json:"limit,omitempty"`
	Cursor                string `json:"cursor,omitempty"`
	NextCursor            string `json:"next_cursor,omitempty"`
	HasMore               bool   `json:"has_more"`
	DefaultSelectedBikeID string `json:"default_selected_bike_id,omitempty"`
}

type BikeListResponse struct {
	Data []BikeDTO    `json:"data"`
	Meta BikeListMeta `json:"meta"`
}

type DriverContact struct {
	ID             string
	OrganizationID string
	Name           string
	Phone          string
	IDNumber       string
	LicenseNumber  string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
