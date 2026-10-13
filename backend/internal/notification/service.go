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

type Service struct {
	snsClient       SNSClient
	repo            *Repository
	platformAppARN  string
	topicARN        string
}

func NewService(client SNSClient, repo *Repository, platformAppARN, topicARN string) *Service {
	return &Service{
		snsClient:      client,
		repo:           repo,
		platformAppARN: platformAppARN,
		topicARN:       topicARN,
	}
}

func (s *Service) RegisterDevice(ctx context.Context, userID, token, platform, phoneNumber string) (string, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return "", fmt.Errorf("device token is required")
	}
	platform = strings.ToUpper(strings.TrimSpace(platform))
	if platform == "" {
		platform = "GCM"
	}

	appARN := s.platformAppARN
	if appARN == "" {
		appARN = "arn:aws:sns:local:123456789012:app/" + platform + "/motorguard-app"
	}

	endpointARN, err := s.snsClient.CreatePlatformEndpoint(ctx, appARN, token, userID)
	if err != nil {
		slog.Error("Failed to create SNS platform endpoint", "error", err, "user_id", userID)
		return "", err
	}

	ep := &DeviceEndpoint{
		ID:          uuid.NewString(),
		UserID:      userID,
		EndpointARN: endpointARN,
		DeviceToken: token,
		Platform:    platform,
		PhoneNumber: strings.TrimSpace(phoneNumber),
	}

	if err := s.repo.SaveEndpoint(ctx, ep); err != nil {
		slog.Warn("Failed to persist endpoint in repo", "error", err, "endpoint_arn", endpointARN)
	}

	slog.Info("Successfully registered device for SNS push notifications",
		"user_id", userID,
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

	// Prepare FCM Payload for mobile/browser
	fcmPayloadMap := map[string]any{
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
	fcmBytes, _ := json.Marshal(fcmPayloadMap)
	fcmPayload := string(fcmBytes)

	// 1. Broadcast to user's registered SNS push endpoints
	endpoints, err := s.repo.GetEndpointsByUserID(ctx, userID)
	if err == nil && len(endpoints) > 0 {
		for _, ep := range endpoints {
			_, pubErr := s.snsClient.PublishToEndpoint(ctx, ep.EndpointARN, alert.Title, alert.Body, fcmPayload)
			if pubErr != nil {
				slog.Warn("Failed to push to SNS endpoint", "endpoint_arn", ep.EndpointARN, "error", pubErr)
			} else {
				slog.Info("SNS Push dispatched successfully", "endpoint_arn", ep.EndpointARN, "title", alert.Title)
			}

			// If critical alert and user has emergency phone registered, send SMS via SNS
			isCritical := isCriticalEvent(alert.EventType)
			if isCritical && ep.PhoneNumber != "" {
				smsMsg := fmt.Sprintf("🚨 MOTORGUARD ALERT: %s. %s", alert.Title, alert.Body)
				_, smsErr := s.snsClient.PublishSMS(ctx, ep.PhoneNumber, smsMsg)
				if smsErr != nil {
					slog.Warn("Failed to send emergency SMS via SNS", "phone", ep.PhoneNumber, "error", smsErr)
				} else {
					slog.Info("Emergency SMS dispatched via SNS", "phone", ep.PhoneNumber)
				}
			}
		}
	}

	// 2. Publish to SNS Global Topic (if configured)
	if s.topicARN != "" {
		_, topicErr := s.snsClient.PublishToTopic(ctx, s.topicARN, alert.Title, alert.Body, fcmPayload)
		if topicErr != nil {
			slog.Warn("Failed to publish to SNS topic", "topic_arn", s.topicARN, "error", topicErr)
		}
	}

	return nil
}

func (s *Service) SendTestNotification(ctx context.Context, userID, title, body, phoneNumber, bikeID string) error {
	if title == "" {
		title = "MotorGuard SNS Test Alert"
	}
	if body == "" {
		body = "Push notification channel verified! You will receive instant motorcycle security alerts."
	}

	alert := SecurityAlert{
		BikeID:     bikeID,
		Title:      title,
		Body:       body,
		EventType:  "TEST_ALERT",
		RecordedAt: time.Now().UTC(),
	}

	if phoneNumber != "" {
		smsMsg := fmt.Sprintf("MotorGuard Alert Test: %s. SMS delivery verified.", body)
		_, _ = s.snsClient.PublishSMS(ctx, phoneNumber, smsMsg)
	}

	return s.DispatchSecurityAlert(ctx, userID, alert)
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
