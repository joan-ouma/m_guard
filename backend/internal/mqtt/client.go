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
	"motoguard-backend/internal/service"

	pahomqtt "github.com/eclipse/paho.mqtt.golang"
)

type MQTTSubscriber struct {
	client           pahomqtt.Client
	telemetryService *service.TelemetryService
	topic            string
}

func NewMQTTSubscriber(brokerURL string, clientID string, telemetryService *service.TelemetryService) (*MQTTSubscriber, error) {
	topicTelemetry := "motoguard/v1/telemetry/+"
	topicEvents := "motoguard/v1/events/+"

	opts := pahomqtt.NewClientOptions()
	opts.AddBroker(brokerURL)
	opts.SetClientID(clientID)
	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(3 * time.Second)
	opts.SetKeepAlive(30 * time.Second)

	handler := func(c pahomqtt.Client, m pahomqtt.Message) {
		if strings.HasPrefix(m.Topic(), "motoguard/v1/events/") {
			var evPayload struct {
				SensorID  string  `json:"sensor_id"`
				EventType string  `json:"event_type"`
				Event     string  `json:"event"`
				Lat       float64 `json:"lat"`
				Latitude  float64 `json:"latitude"`
				Lng       float64 `json:"lng"`
				Longitude float64 `json:"longitude"`
				Timestamp int64   `json:"ts"`
			}
			if err := json.Unmarshal(m.Payload(), &evPayload); err != nil {
				log.Printf("invalid mqtt event payload: %v", err)
				return
			}
			sensorID := strings.TrimSpace(evPayload.SensorID)
			if sensorID == "" {
				parts := strings.Split(m.Topic(), "/")
				if len(parts) > 0 {
					sensorID = parts[len(parts)-1]
				}
			}
			evType := evPayload.EventType
			if evType == "" {
				evType = evPayload.Event
			}
			lat := evPayload.Lat
			if lat == 0 {
				lat = evPayload.Latitude
			}
			lng := evPayload.Lng
			if lng == 0 {
				lng = evPayload.Longitude
			}
			var recTime time.Time
			if evPayload.Timestamp > 0 {
				recTime = time.Unix(evPayload.Timestamp, 0).UTC()
			} else {
				recTime = time.Now().UTC()
			}
			ingestCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := telemetryService.IngestEvent(ingestCtx, sensorID, evType, lat, lng, recTime); err != nil {
				log.Printf("failed to ingest event for sensor %s: %v", sensorID, err)
			} else {
				log.Printf("[MQTT] Successfully recorded event %s for sensor %s", evType, sensorID)
			}
			return
		}

		var payload domain.TelemetryPayload
		if err := json.Unmarshal(m.Payload(), &payload); err != nil {
			log.Printf("invalid mqtt json payload: %v", err)
			return
		}

		if strings.TrimSpace(payload.SensorID) == "" {
			parts := strings.Split(m.Topic(), "/")
			if len(parts) > 0 {
				payload.SensorID = parts[len(parts)-1]
			}
		}

		log.Printf("[MQTT] Received telemetry for sensor %s: ign=%v, spd=%.1f", payload.SensorID, payload.Ignition, payload.SpeedKMH)

		ingestCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := telemetryService.IngestTelemetry(ingestCtx, payload); err != nil {
			log.Printf("failed to ingest telemetry for sensor %s: %v", payload.SensorID, err)
		} else {
			log.Printf("[MQTT] Successfully ingested telemetry for sensor %s", payload.SensorID)
		}
	}

	opts.SetOnConnectHandler(func(c pahomqtt.Client) {
		log.Printf("MQTT connected / reconnected. Subscribing to topics: %s, %s", topicTelemetry, topicEvents)
		c.Subscribe(topicTelemetry, 1, handler)
		c.Subscribe(topicEvents, 1, handler)
	})

	client := pahomqtt.NewClient(opts)

	return &MQTTSubscriber{
		client:           client,
		telemetryService: telemetryService,
		topic:            topicTelemetry,
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
		s.client.Unsubscribe(s.topic, "motoguard/v1/events/+")
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
