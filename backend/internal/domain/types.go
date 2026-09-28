package domain

import (
	"encoding/json"
	"net/http"
	"time"
)

type User struct {
	ID           string
	Email        string
	PasswordHash string
	Name         string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Session struct {
	ID               string
	UserID           string
	RefreshTokenHash string
	ExpiresAt        time.Time
	RevokedAt        time.Time
	UserAgent        string
	IPAddress        string
	CreatedAt        time.Time
}

type PasswordResetToken struct {
	ID        string
	UserID    string
	TokenHash string
	ExpiresAt time.Time
	UsedAt    time.Time
	CreatedAt time.Time
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UserDTO struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  string `json:"role,omitempty"`
}

type SessionData struct {
	AccessToken string  `json:"access_token"`
	TokenType   string  `json:"token_type"`
	ExpiresIn   int64   `json:"expires_in"`
	User        UserDTO `json:"user"`
}

type AuthResponse struct {
	Data SessionData `json:"data"`
}

type PasswordResetRequestDTO struct {
	Email string `json:"email"`
}

type PasswordResetConfirmDTO struct {
	Password string `json:"password"`
}

type MessageResponse struct {
	Message string `json:"message"`
}

type ProblemDetails struct {
	Type     string              `json:"type,omitempty"`
	Title    string              `json:"title"`
	Status   int                 `json:"status"`
	Detail   string              `json:"detail,omitempty"`
	Instance string              `json:"instance,omitempty"`
	Errors   map[string][]string `json:"errors,omitempty"`
}

func (p ProblemDetails) Error() string {
	return p.Detail
}

func (p ProblemDetails) WriteJSON(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(p.Status)
	_ = json.NewEncoder(w).Encode(p)
}

func NewUnauthorizedError(detail string, instance string) ProblemDetails {
	return ProblemDetails{
		Type:     "https://motoguard.co.ke/errors/unauthorized",
		Title:    "Unauthorized",
		Status:   http.StatusUnauthorized,
		Detail:   detail,
		Instance: instance,
	}
}

func NewBadRequestError(detail string, instance string, errors map[string][]string) ProblemDetails {
	return ProblemDetails{
		Type:     "https://motoguard.co.ke/errors/bad-request",
		Title:    "Bad Request",
		Status:   http.StatusBadRequest,
		Detail:   detail,
		Instance: instance,
		Errors:   errors,
	}
}

func NewNotFoundError(detail string, instance string) ProblemDetails {
	return ProblemDetails{
		Type:     "https://motoguard.co.ke/errors/not-found",
		Title:    "Not Found",
		Status:   http.StatusNotFound,
		Detail:   detail,
		Instance: instance,
	}
}

func NewTooManyRequestsError(detail string, instance string) ProblemDetails {
	return ProblemDetails{
		Type:     "https://motoguard.co.ke/errors/too-many-requests",
		Title:    "Too Many Requests",
		Status:   http.StatusTooManyRequests,
		Detail:   detail,
		Instance: instance,
	}
}

func NewInternalServerError(instance string) ProblemDetails {
	return ProblemDetails{
		Type:     "https://motoguard.co.ke/errors/internal-server-error",
		Title:    "Internal Server Error",
		Status:   http.StatusInternalServerError,
		Detail:   "Internal server error",
		Instance: instance,
	}
}

func NewForbiddenError(detail string, instance string) ProblemDetails {
	return ProblemDetails{
		Type:     "https://motoguard.co.ke/errors/forbidden",
		Title:    "Forbidden",
		Status:   http.StatusForbidden,
		Detail:   detail,
		Instance: instance,
	}
}

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
	Action string `json:"action"` // "IMMOBILIZE" or "REACTIVATE"
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
	RegistrationNumber string  `json:"registration_number"`
	Label              string  `json:"label"`
	Make               string  `json:"make"`
	Model              string  `json:"model"`
	Year               int     `json:"year"`
	Color              string  `json:"color"`
	CurrentAddress     string  `json:"current_address"`
	SubCounty          string  `json:"sub_county"`
	County             string  `json:"county"`
	Status             string  `json:"status"`
	GeofenceID         string  `json:"geofence_id"`
	DriverContactID    string  `json:"driver_contact_id"`
}

type BikeListMeta struct {
	Total                 int    `json:"total"`
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

type TelemetryPayload struct {
	SensorID       string  `json:"sensor_id"`
	Latitude       float64 `json:"lat"`
	Longitude      float64 `json:"lng"`
	Timestamp      int64   `json:"ts"`
	SpeedKMH       float64 `json:"spd"`
	HeadingDeg     float64 `json:"hdg"`
	Ignition       bool    `json:"ign"`
	BatteryPercent int     `json:"bat"`
	SignalQuality  int     `json:"csq"`
	Event          string  `json:"event,omitempty"`
	EventType      string  `json:"event_type,omitempty"`
}

type BreadcrumbDTO struct {
	ID             int64     `json:"id"`
	BikeID         string    `json:"bike_id"`
	Latitude       float64   `json:"latitude"`
	Longitude      float64   `json:"longitude"`
	SpeedKMH       float64   `json:"speed_kmh"`
	HeadingDeg     float64   `json:"heading_deg"`
	Ignition       bool      `json:"ignition"`
	BatteryPercent int       `json:"battery_percent"`
	Address        string    `json:"address"`
	GeofenceBreach bool      `json:"geofence_breach"`
	RecordedAt     time.Time `json:"recorded_at"`
}

type BikeEventDTO struct {
	ID         int64     `json:"id"`
	BikeID     string    `json:"bike_id"`
	EventType  string    `json:"event_type"`
	Latitude   float64   `json:"latitude"`
	Longitude  float64   `json:"longitude"`
	RecordedAt time.Time `json:"recorded_at"`
}




