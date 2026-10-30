package notification

import (
	"context"
	"time"
)

type SecurityAlert struct {
	BikeID     string    `json:"bike_id"`
	Title      string    `json:"title"`
	Body       string    `json:"body"`
	EventType  string    `json:"event_type"`
	Latitude   float64   `json:"latitude,omitempty"`
	Longitude  float64   `json:"longitude,omitempty"`
	RecordedAt time.Time `json:"recorded_at"`
}

type DeviceEndpoint struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	EndpointARN string    `json:"endpoint_arn"`
	DeviceToken string    `json:"device_token"`
	Platform    string    `json:"platform"` // "GCM", "APNS", "WEB", etc.
	PhoneNumber string    `json:"phone_number,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type RegisterDeviceRequest struct {
	Token       string `json:"token"`
	Platform    string `json:"platform"`
	PhoneNumber string `json:"phone_number,omitempty"`
}

type RegisterDeviceResponse struct {
	EndpointARN string `json:"endpoint_arn"`
	Message     string `json:"message"`
}

type TestNotificationRequest struct {
	Title       string `json:"title,omitempty"`
	Body        string `json:"body,omitempty"`
	PhoneNumber string `json:"phone_number,omitempty"`
	BikeID      string `json:"bike_id,omitempty"`
}

type SNSClient interface {
	CreatePlatformEndpoint(ctx context.Context, platformAppARN, token, customUserData string) (string, error)
	DeleteEndpoint(ctx context.Context, endpointARN string) error
	PublishToEndpoint(ctx context.Context, endpointARN string, subject, rawMessage, gcmPayload string) (string, error)
	PublishToTopic(ctx context.Context, topicARN string, subject, rawMessage, gcmPayload string) (string, error)
}
