package domain

import "time"

// Recovery: a theft report a rider files after losing a bike, and the contacts they keep for it.

type TheftIncident struct {
	ID             string
	UserID         string
	BikeID         string
	Mode           string
	LostAt         time.Time
	LocationLost   string
	Latitude       *float64
	Longitude      *float64
	OBNumber       string
	AttachmentName string
	CreatedAt      time.Time
}

type IncidentAttachment struct {
	FileName    string
	ContentType string
	Data        []byte
}

type TheftIncidentDTO struct {
	ID             string    `json:"id"`
	BikeID         string    `json:"bike_id,omitempty"`
	Mode           string    `json:"mode"`
	LostAt         time.Time `json:"lost_at"`
	LocationLost   string    `json:"location_lost"`
	Latitude       *float64  `json:"latitude"`
	Longitude      *float64  `json:"longitude"`
	OBNumber       string    `json:"ob_number,omitempty"`
	AttachmentName string    `json:"attachment_name,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

type CreateIncidentRequest struct {
	BikeID       string   `json:"bike_id"`
	Mode         string   `json:"mode"`
	LostAt       string   `json:"lost_at"`
	LocationLost string   `json:"location_lost"`
	Latitude     *float64 `json:"latitude"`
	Longitude    *float64 `json:"longitude"`
}

type UserContact struct {
	ID        string
	UserID    string
	Name      string
	Phone     string
	Category  string
	CreatedAt time.Time
}

type UserContactDTO struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Phone     string    `json:"phone"`
	Category  string    `json:"category"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateContactRequest struct {
	Name     string `json:"name"`
	Phone    string `json:"phone"`
	Category string `json:"category"`
}
