package domain

import "time"

// UserExport is everything the platform holds about one rider, for the rider to download.
// It never includes the password hash, sessions, or device push tokens.
type UserExport struct {
	ExportedAt        time.Time             `json:"exported_at"`
	Profile           ExportProfile         `json:"profile"`
	Bikes             []ExportBike          `json:"bikes"`
	LocationHistory   []ExportLocation      `json:"location_history_last_90_days"`
	BikeEvents        []ExportBikeEvent     `json:"bike_events"`
	TheftReports      []ExportTheftReport   `json:"theft_reports"`
	Contacts          []ExportContact       `json:"contacts"`
	DriverContacts    []ExportDriverContact `json:"driver_contacts"`
	AssistantMessages []ExportChatMessage   `json:"assistant_history"`
}

type ExportProfile struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	Name        string    `json:"name"`
	PhoneNumber string    `json:"phone_number"`
	NationalID  string    `json:"national_id"`
	CreatedAt   time.Time `json:"created_at"`
}

type ExportBike struct {
	ID                 string    `json:"id"`
	RegistrationNumber string    `json:"registration_number"`
	Label              string    `json:"label"`
	Make               string    `json:"make"`
	Model              string    `json:"model"`
	Year               int       `json:"year"`
	Color              string    `json:"color"`
	IMEI               string    `json:"imei"`
	CreatedAt          time.Time `json:"created_at"`
}

type ExportLocation struct {
	BikeID     string    `json:"bike_id"`
	Latitude   float64   `json:"latitude"`
	Longitude  float64   `json:"longitude"`
	SpeedKMH   float64   `json:"speed_kmh"`
	Ignition   bool      `json:"ignition"`
	RecordedAt time.Time `json:"recorded_at"`
}

type ExportBikeEvent struct {
	BikeID     string    `json:"bike_id"`
	EventType  string    `json:"event_type"`
	Latitude   float64   `json:"latitude"`
	Longitude  float64   `json:"longitude"`
	RecordedAt time.Time `json:"recorded_at"`
}

type ExportTheftReport struct {
	ID           string     `json:"id"`
	BikeID       string     `json:"bike_id"`
	Mode         string     `json:"mode"`
	LostAt       time.Time  `json:"lost_at"`
	LocationLost string     `json:"location_lost"`
	OBNumber     string     `json:"ob_number"`
	ResolvedAt   *time.Time `json:"resolved_at"`
	CreatedAt    time.Time  `json:"created_at"`
}

type ExportContact struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Phone     string    `json:"phone"`
	Category  string    `json:"category"`
	CreatedAt time.Time `json:"created_at"`
}

type ExportDriverContact struct {
	Name      string    `json:"name"`
	Phone     string    `json:"phone"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

type ExportChatMessage struct {
	Role      string    `json:"role"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}
