package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"motoguard-backend/internal/africastalking"

	"github.com/google/uuid"
)

type RegisterDeviceParams struct {
	UserID      string
	Token       string
	Platform    string
	PhoneNumber string
}

type SendTestAlertParams struct {
	UserID      string
	Title       string
	Body        string
	PhoneNumber string
	BikeID      string
}

type Service struct {
	snsClient      SNSClient
	atClient       africastalking.Client
	repo           *Repository
	platformAppARN string
	topicARN       string
}

func NewService(client SNSClient, repo *Repository, platformAppARN, topicARN string) *Service {
	return &Service{
		snsClient:      client,
		repo:           repo,
		platformAppARN: platformAppARN,
		topicARN:       topicARN,
	}
}

func (s *Service) SetATClient(client africastalking.Client) {
	s.atClient = client
}

func (s *Service) RegisterDevice(ctx context.Context, params RegisterDeviceParams) (string, error) {
	token := strings.TrimSpace(params.Token)
	if token == "" {
		return "", fmt.Errorf("device token is required")
	}

	platform := strings.ToUpper(strings.TrimSpace(params.Platform))
	if platform == "" {
		platform = "GCM"
	}

	appARN := s.platformAppARN
	if appARN == "" {
		return "", fmt.Errorf("AWS SNS Platform Application ARN is not configured")
	}

	endpointARN, err := s.snsClient.CreatePlatformEndpoint(ctx, appARN, token, params.UserID)
	if err != nil {
		slog.Error("Failed to create SNS platform endpoint", "error", err, "user_id", params.UserID)
		return "", err
	}

	ep := &DeviceEndpoint{
		ID:          uuid.NewString(),
		UserID:      params.UserID,
		EndpointARN: endpointARN,
		DeviceToken: token,
		Platform:    platform,
		PhoneNumber: strings.TrimSpace(params.PhoneNumber),
	}

	err = s.repo.SaveEndpoint(ctx, ep)
	if err != nil {
		slog.Warn("Failed to persist endpoint in repo", "error", err, "endpoint_arn", endpointARN)
	}

	slog.Info("Successfully registered device for SNS push notifications",
		"user_id", params.UserID,
		"endpoint_arn", endpointARN,
		"platform", platform,
	)

	return endpointARN, nil
}

func (s *Service) UnregisterDevice(ctx context.Context, endpointARN string) error {
	_ = s.snsClient.DeleteEndpoint(ctx, endpointARN)
	return s.repo.DeleteEndpointByARN(ctx, endpointARN)
}

func (s *Service) DispatchSecurityAlert(ctx context.Context, userID string, alert SecurityAlert) error {
	if alert.RecordedAt.IsZero() {
		alert.RecordedAt = time.Now().UTC()
	}

	fcmPayload := s.buildFCMPayload(alert)
	s.broadcastToUserDevices(ctx, userID, alert, fcmPayload)

	return nil
}

func (s *Service) buildFCMPayload(alert SecurityAlert) string {
	payload := map[string]any{
		"notification": map[string]string{
			"title": alert.Title,
			"body":  alert.Body,
			"icon":  "/favicon.svg",
		},
		"data": map[string]any{
			"bike_id":     alert.BikeID,
			"event_type":  alert.EventType,
			"latitude":    fmt.Sprintf("%f", alert.Latitude),
			"longitude":   fmt.Sprintf("%f", alert.Longitude),
			"recorded_at": alert.RecordedAt.Format(time.RFC3339),
			"url":         fmt.Sprintf("/history?bike=%s", alert.BikeID),
		},
	}
	bytes, _ := json.Marshal(payload)
	return string(bytes)
}

func (s *Service) broadcastToUserDevices(ctx context.Context, userID string, alert SecurityAlert, fcmPayload string) {
	endpoints, err := s.repo.GetEndpointsByUserID(ctx, userID)
	if err != nil || len(endpoints) == 0 {
		return
	}

	slog.Info("Broadcasting alert to all devices under account", "user_id", userID, "device_count", len(endpoints))

	for _, ep := range endpoints {
		s.dispatchToEndpoint(ctx, ep, alert, fcmPayload)
	}
}

func (s *Service) dispatchToEndpoint(ctx context.Context, ep DeviceEndpoint, alert SecurityAlert, fcmPayload string) {
	_, err := s.snsClient.PublishToEndpoint(ctx, ep.EndpointARN, alert.Title, alert.Body, fcmPayload)
	if err == nil {
		slog.Info("SNS Push dispatched successfully to device", "endpoint_arn", ep.EndpointARN, "device_id", ep.ID, "title", alert.Title)
		return
	}

	slog.Warn("Failed to push to SNS endpoint", "endpoint_arn", ep.EndpointARN, "device_id", ep.ID, "error", err)
	if isStaleEndpointError(err) {
		_ = s.repo.DeleteEndpointByARN(ctx, ep.EndpointARN)
	}
}

func (s *Service) SendTestNotification(ctx context.Context, params SendTestAlertParams) error {
	title := params.Title
	if title == "" {
		title = "MotorGuard SNS Test Alert"
	}

	body := params.Body
	if body == "" {
		body = "Push notification channel verified! You will receive instant motorcycle security alerts."
	}

	alert := SecurityAlert{
		BikeID:     params.BikeID,
		Title:      title,
		Body:       body,
		EventType:  "TEST_ALERT",
		RecordedAt: time.Now().UTC(),
	return s.DispatchSecurityAlert(ctx, params.UserID, alert)
}

func isCriticalEvent(eventType string) bool {
	upper := strings.ToUpper(eventType)
	return strings.Contains(upper, "BREACH") ||
		strings.Contains(upper, "TAMPER") ||
		strings.Contains(upper, "THEFT") ||
		strings.Contains(upper, "DURESS") ||
		strings.Contains(upper, "CRASH") ||
		strings.Contains(upper, "IMMOBILIZE")
}

func isStaleEndpointError(err error) bool {
	if err == nil {
		return false
	}
	errMsg := err.Error()
	return strings.Contains(errMsg, "EndpointDisabled") || strings.Contains(errMsg, "NotFound")
}
