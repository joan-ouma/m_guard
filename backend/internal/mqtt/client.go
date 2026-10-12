package mqtt

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"motoguard-backend/internal/domain"
	"motoguard-backend/internal/geolocation"
	"motoguard-backend/internal/service"

	pahomqtt "github.com/eclipse/paho.mqtt.golang"
)

type MQTTSubscriber struct {
	client           pahomqtt.Client
	telemetryService *service.TelemetryService
	cellResolver     *geolocation.CellResolver
	topics           []string
}

func NewMQTTSubscriber(brokerURL string, clientID string, username string, password string, telemetryService *service.TelemetryService) (*MQTTSubscriber, error) {
	topics := []string{
		"motoguard/v1/#",
	}

	cellResolver := geolocation.NewCellResolver()

	opts := pahomqtt.NewClientOptions()
	opts.AddBroker(brokerURL)
	opts.SetClientID(fmt.Sprintf("%s-%d", clientID, time.Now().UnixNano()%100000))
	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(3 * time.Second)
	opts.SetKeepAlive(30 * time.Second)

	opts.SetConnectionLostHandler(func(c pahomqtt.Client, err error) {
		log.Printf("[MQTT] Connection lost: %v", err)
	})

	if username != "" {
		opts.SetUsername(username)
		if password != "" {
			opts.SetPassword(password)
		}
	}

	handler := func(c pahomqtt.Client, m pahomqtt.Message) {
		topic := m.Topic()
		log.Printf("[MQTT Incoming] Topic: %s, Size: %d bytes, Payload: %s", topic, len(m.Payload()), string(m.Payload()))

		parts := strings.Split(topic, "/")
		sensorID := ""
		if len(parts) > 0 {
			sensorID = strings.TrimSpace(parts[len(parts)-1])
		}

		// Handle Burglary & Tamper Alert (Instant Trigger)
		if strings.Contains(topic, "/alert") || strings.Contains(topic, "/events") {
			var alert domain.AlertPayload
			if err := json.Unmarshal(m.Payload(), &alert); err != nil {
				log.Printf("[MQTT Alert] invalid json payload from %s: %v", topic, err)
				return
			}

			// When GPS is 0,0, resolve location from Cell Tower database
			if alert.Latitude == 0 && alert.Longitude == 0 && alert.MCC > 0 && alert.LAC > 0 && alert.CID > 0 {
				geoCtx, geoCancel := context.WithTimeout(context.Background(), 4*time.Second)
				if lat, lng, ok := cellResolver.Resolve(geoCtx, alert.MCC, alert.MNC, alert.LAC, alert.CID); ok {
					alert.Latitude = lat
					alert.Longitude = lng
					alert.LocType = "lbs"
				}
				geoCancel()
			}

			recTime := time.Now().UTC()
			if alert.Timestamp > 0 {
				recTime = time.Unix(alert.Timestamp, 0).UTC()
			}

			reason := strings.TrimSpace(alert.Reason)
			if reason == "" {
				reason = "BURGLARY_TAMPER_ALERT"
			}

			ingestCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			// 1. Record the burglary/tamper alert event
			if err := telemetryService.IngestEvent(ingestCtx, sensorID, reason, alert.Latitude, alert.Longitude, recTime); err != nil {
				log.Printf("[MQTT Alert] failed to ingest event %s for sensor %s: %v", reason, sensorID, err)
			} else {
				log.Printf("[MQTT Alert] Recorded alert [%s] for sensor %s at (%.6f, %.6f)", reason, sensorID, alert.Latitude, alert.Longitude)
			}

			// 2. Ingest telemetry so live map position & status update immediately
			telemetryPayload := domain.TelemetryPayload{
				SensorID:       sensorID,
				Latitude:       alert.Latitude,
				Longitude:      alert.Longitude,
				LocType:        alert.LocType,
				MCC:            alert.MCC,
				MNC:            alert.MNC,
				LAC:            alert.LAC,
				CID:            alert.CID,
				SignalQuality:  alert.SignalQuality,
				RSSIDBm:        alert.RSSIDBm,
				SpeedKMH:       alert.SpeedKMH,
				HeadingDeg:     alert.HeadingDeg,
				Ignition:       alert.Ignition,
				BatteryPercent: alert.BatteryPercent,
				Timestamp:      recTime.Unix(),
				Alert:          alert.Alert,
				Reason:         alert.Reason,
			}
			_ = telemetryService.IngestTelemetry(ingestCtx, telemetryPayload)
			return
		}

		// Handle Periodic Telemetry (Normal Tracking)
		var payload domain.TelemetryPayload
		if err := json.Unmarshal(m.Payload(), &payload); err != nil {
			log.Printf("[MQTT Telemetry] invalid json payload from %s: %v", topic, err)
			return
		}

		if strings.TrimSpace(payload.SensorID) == "" {
			payload.SensorID = sensorID
		}

		// When GPS is 0,0, resolve location from Cell Tower database
		if payload.Latitude == 0 && payload.Longitude == 0 && payload.MCC > 0 && payload.LAC > 0 && payload.CID > 0 {
			geoCtx, geoCancel := context.WithTimeout(context.Background(), 4*time.Second)
			if lat, lng, ok := cellResolver.Resolve(geoCtx, payload.MCC, payload.MNC, payload.LAC, payload.CID); ok {
				payload.Latitude = lat
				payload.Longitude = lng
				payload.LocType = "lbs"
			}
			geoCancel()
		}

		recTime := time.Now().UTC()
		if payload.Timestamp > 0 {
			recTime = time.Unix(payload.Timestamp, 0).UTC()
		}

		ingestCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := telemetryService.IngestTelemetry(ingestCtx, payload); err != nil {
			log.Printf("[MQTT Telemetry] failed to ingest telemetry for sensor %s: %v", payload.SensorID, err)
		} else {
			log.Printf("[MQTT Telemetry] Ingested sensor %s (loc=%s, lat=%.6f, lng=%.6f, ign=%v, spd=%.1f)",
				payload.SensorID, payload.LocType, payload.Latitude, payload.Longitude, payload.Ignition, payload.SpeedKMH)
		}

		// If telemetry also includes an active alert flag or reason
		if payload.Alert && payload.Reason != "" {
			_ = telemetryService.IngestEvent(ingestCtx, payload.SensorID, payload.Reason, payload.Latitude, payload.Longitude, recTime)
		}
	}

	opts.SetOnConnectHandler(func(c pahomqtt.Client) {
		log.Printf("MQTT connected / reconnected. Subscribing to topics: %s", strings.Join(topics, ", "))
		for _, top := range topics {
			c.Subscribe(top, 1, handler)
		}
	})

	client := pahomqtt.NewClient(opts)

	return &MQTTSubscriber{
		client:           client,
		telemetryService: telemetryService,
		cellResolver:     cellResolver,
		topics:           topics,
	}, nil
}

func (s *MQTTSubscriber) Start(ctx context.Context) error {
	token := s.client.Connect()
	if token.Wait() && token.Error() != nil {
		return token.Error()
	}

	return nil
}

func (s *MQTTSubscriber) Stop() {
	if s.client.IsConnected() {
		for _, top := range s.topics {
			s.client.Unsubscribe(top)
		}
		s.client.Disconnect(250)
	}
}

func (s *MQTTSubscriber) PublishCommand(imei string, action string) error {
	if !s.client.IsConnected() {
		return errors.New("mqtt client is not connected")
	}
	topic := fmt.Sprintf("motoguard/v1/commands/%s", imei)
	payload, err := json.Marshal(map[string]any{
		"action":    action,
		"timestamp": time.Now().Unix(),
	})
	if err != nil {
		return err
	}
	token := s.client.Publish(topic, 1, false, payload)
	token.Wait()
	log.Printf("[MQTT] Published command %s to topic %s", action, topic)
	return token.Error()
}
