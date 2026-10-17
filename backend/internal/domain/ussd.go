package domain

import (
	"time"
)

// USSDRequest holds inbound payload from Africa's Talking USSD Gateway.
type USSDRequest struct {
	SessionID   string `json:"sessionId"`
	ServiceCode string `json:"serviceCode"`
	PhoneNumber string `json:"phoneNumber"`
	Text        string `json:"text"`
}

// SMSIncomingRequest represents inbound SMS payload from Africa's Talking SMS Webhook.
type SMSIncomingRequest struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Text   string `json:"text"`
	Date   string `json:"date"`
	ID     string `json:"id"`
	LinkID string `json:"linkId"`
}

// RiderBikeInfo represents motorcycle details relevant to USSD and SMS interactions.
type RiderBikeInfo struct {
	ID                 string     `json:"id"`
	UserID             string     `json:"user_id"`
	RegistrationNumber string     `json:"registration_number"`
	Label              string     `json:"label"`
	Make               string     `json:"make"`
	Model              string     `json:"model"`
	Status             string     `json:"status"`
	Ignition           bool       `json:"ignition"`
	SpeedKMH           float64    `json:"speed_kmh"`
	Latitude           float64    `json:"latitude"`
	Longitude          float64    `json:"longitude"`
	CurrentAddress     string     `json:"current_address"`
	SubCounty          string     `json:"sub_county"`
	County             string     `json:"county"`
	BatteryPercent     int        `json:"battery_percent"`
	SignalQualityCSQ   int        `json:"signal_quality_csq"`
	IsImmobilized      bool       `json:"is_immobilized"`
	IMEI               string     `json:"imei"`
	LastTelemetryAt    *time.Time `json:"last_telemetry_at,omitempty"`
	OwnerPhone         string     `json:"owner_phone,omitempty"`
	OwnerName          string     `json:"owner_name,omitempty"`
	RiderName          string     `json:"rider_name,omitempty"`
}

// RiderPIN holds PIN authentication state for a phone number.
type RiderPIN struct {
	PhoneNumber    string     `json:"phone_number"`
	PINHash        string     `json:"-"`
	FailedAttempts int        `json:"failed_attempts"`
	LockedUntil    *time.Time `json:"locked_until,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}
