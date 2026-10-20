package mqtt

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
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

func resolveCellPosition(ctx context.Context, resolver *geolocation.CellResolver, mcc, mnc, lac, cid, rssi int, cells []domain.CellTower) (float64, float64, float64, bool) {
	if resolver == nil || mcc <= 0 {
		return 0, 0, 0, false
	}
	towers := cells
	if len(towers) == 0 && lac > 0 && cid > 0 {
		towers = []domain.CellTower{{LAC: lac, CID: cid, RSSIDBm: rssi}}
	}
	if len(towers) == 0 {
		return 0, 0, 0, false
	}
	return resolver.ResolveMulti(ctx, mcc, mnc, towers)
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
		slog.Warn("MQTT connection lost", "error", err)
	})

	if username != "" {
		opts.SetUsername(username)
		if password != "" {
			opts.SetPassword(password)
		}
	}

	handler := func(c pahomqtt.Client, m pahomqtt.Message) {
		topic := m.Topic()
		rawPayload := string(m.Payload())
		slog.Info("MQTT incoming message", "topic", topic, "bytes", len(m.Payload()), "raw", rawPayload)

		parts := strings.Split(topic, "/")
		sensorID := ""
		if len(parts) > 0 {
			sensorID = strings.TrimSpace(parts[len(parts)-1])
		}

		// Handle Burglary & Tamper Alert (Instant Trigger)
		if strings.Contains(topic, "/alert") || strings.Contains(topic, "/events") {
			var alert domain.AlertPayload
			if err := json.Unmarshal(m.Payload(), &alert); err != nil {
				slog.Error("MQTT alert invalid json payload", "topic", topic, "error", err, "raw", rawPayload)
				return
			}

			// When GPS is 0,0, resolve location from Multi-Cell triangulation or serving cell
			if alert.Latitude == 0 && alert.Longitude == 0 {
				geoCtx, geoCancel := context.WithTimeout(context.Background(), 4*time.Second)
				lat, lng, acc, ok := resolveCellPosition(geoCtx, cellResolver, alert.MCC, alert.MNC, alert.LAC, alert.CID, alert.RSSIDBm, alert.Cells)
				geoCancel()
				slog.Info("LBS alert resolved", "sensor_id", sensorID, "mcc", alert.MCC, "mnc", alert.MNC, "lac", alert.LAC, "cid", alert.CID, "lat", lat, "lng", lng, "accuracy", acc, "ok", ok)
				if ok {
					alert.Latitude = lat
					alert.Longitude = lng
					alert.Accuracy = acc
					alert.LocType = "lbs"
				}
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
				slog.Error("MQTT alert failed to ingest event", "reason", reason, "sensor_id", sensorID, "error", err)
			} else {
				slog.Info("MQTT alert recorded", "reason", reason, "sensor_id", sensorID, "lat", alert.Latitude, "lng", alert.Longitude)
			}

			// 2. Ingest telemetry so live map position & status update immediately
			telemetryPayload := domain.TelemetryPayload{
				SensorID:       sensorID,
				Latitude:       alert.Latitude,
				Longitude:      alert.Longitude,
				Accuracy:       alert.Accuracy,
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
			slog.Error("MQTT telemetry invalid json payload", "topic", topic, "error", err, "raw", rawPayload)
			return
		}

		if strings.TrimSpace(payload.SensorID) == "" {
			payload.SensorID = sensorID
		}

		// When GPS is 0,0, resolve location from Multi-Cell triangulation or serving cell
		if payload.Latitude == 0 && payload.Longitude == 0 {
			geoCtx, geoCancel := context.WithTimeout(context.Background(), 4*time.Second)
			lat, lng, acc, ok := resolveCellPosition(geoCtx, cellResolver, payload.MCC, payload.MNC, payload.LAC, payload.CID, payload.RSSIDBm, payload.Cells)
			geoCancel()
			slog.Info("LBS position resolved", "sensor_id", payload.SensorID, "mcc", payload.MCC, "mnc", payload.MNC, "lac", payload.LAC, "cid", payload.CID, "lat", lat, "lng", lng, "accuracy", acc, "ok", ok)
			if ok {
				payload.Latitude = lat
				payload.Longitude = lng
				payload.Accuracy = acc
				payload.LocType = "lbs"
			}
		}

		recTime := time.Now().UTC()
		if payload.Timestamp > 0 {
			recTime = time.Unix(payload.Timestamp, 0).UTC()
		}

		ingestCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := telemetryService.IngestTelemetry(ingestCtx, payload); err != nil {
			slog.Error("MQTT telemetry failed to ingest", "sensor_id", payload.SensorID, "error", err)
		} else {
			slog.Info("MQTT telemetry ingested", "sensor_id", payload.SensorID, "loc_type", payload.LocType, "lat", payload.Latitude, "lng", payload.Longitude, "ignition", payload.Ignition, "speed", payload.SpeedKMH, "raw", rawPayload)
		}

		// If telemetry also includes an active alert flag or reason
		if payload.Alert && payload.Reason != "" {
			_ = telemetryService.IngestEvent(ingestCtx, payload.SensorID, payload.Reason, payload.Latitude, payload.Longitude, recTime)
		}
	}

	opts.SetOnConnectHandler(func(c pahomqtt.Client) {
		slog.Info("MQTT connected / reconnected", "topics", topics)
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
	slog.Info("MQTT published command", "action", action, "topic", topic)
	return token.Error()
}
