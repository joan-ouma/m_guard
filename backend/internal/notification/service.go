package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

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
		appARN = "arn:aws:sns:local:123456789012:app/" + platform + "/motorguard-app"
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
	s.publishGlobalTopic(ctx, alert, fcmPayload)

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
	sentPhones := make(map[string]struct{})

	for _, ep := range endpoints {
		s.dispatchToEndpoint(ctx, ep, alert, fcmPayload)
		s.dispatchEmergencySMS(ctx, ep, alert, sentPhones)
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

func (s *Service) dispatchEmergencySMS(ctx context.Context, ep DeviceEndpoint, alert SecurityAlert, sentPhones map[string]struct{}) {
	if !isCriticalEvent(alert.EventType) {
		return
	}
	if ep.PhoneNumber == "" {
		return
	}
	if _, alreadySent := sentPhones[ep.PhoneNumber]; alreadySent {
		return
	}

	sentPhones[ep.PhoneNumber] = struct{}{}
	smsMsg := fmt.Sprintf("🚨 MOTORGUARD ALERT: %s. %s", alert.Title, alert.Body)
	_, err := s.snsClient.PublishSMS(ctx, ep.PhoneNumber, smsMsg)
	if err != nil {
		slog.Warn("Failed to send emergency SMS via SNS", "phone", ep.PhoneNumber, "error", err)
		return
	}

	slog.Info("Emergency SMS dispatched via SNS", "phone", ep.PhoneNumber)
}

func (s *Service) publishGlobalTopic(ctx context.Context, alert SecurityAlert, fcmPayload string) {
	if s.topicARN == "" {
		return
	}

	_, err := s.snsClient.PublishToTopic(ctx, s.topicARN, alert.Title, alert.Body, fcmPayload)
	if err != nil {
		slog.Warn("Failed to publish to SNS topic", "topic_arn", s.topicARN, "error", err)
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
	}

	if params.PhoneNumber != "" {
		smsMsg := fmt.Sprintf("MotorGuard Alert Test: %s. SMS delivery verified.", body)
		_, _ = s.snsClient.PublishSMS(ctx, params.PhoneNumber, smsMsg)
	}

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
