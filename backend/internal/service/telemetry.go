package service

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"strings"
	"time"

	"motoguard-backend/internal/domain"
	"motoguard-backend/internal/geolocation"
	"motoguard-backend/internal/notification"
	"motoguard-backend/internal/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	movingThresholdKMH    = 1.0
	maxPlausibleSpeedKMH  = 180.0
	minTrackedDistanceKM  = 0.005
	maxHeadingDeg         = 360.0
	maxScaledHeadingDeg   = 3600.0 // some devices report heading in tenths of a degree
	earthRadiusKM         = 6371.0
	reverseGeocodeTimeout = 4 * time.Second
	eventGeofenceBreach   = "GEOFENCE_BREACH"
)

var ErrInvalidSensorID = errors.New("sensor id is required")

// securityEvent is everything needed to tell an owner something happened.
type securityEvent struct {
	bikeID       string
	userID       string
	registration string
	eventType    string
	lat, lng     float64
	at           time.Time
}

type TelemetryService struct {
	pool          *pgxpool.Pool
	telemetry     *repository.TelemetryRepository
	bikes         *repository.BikeRepository
	sse           SSEBroker
	geocoder      *geolocation.ReverseGeocoder
	notifications *notification.Service
}

func NewTelemetryService(pool *pgxpool.Pool, telemetry *repository.TelemetryRepository) *TelemetryService {
	return &TelemetryService{
		pool:      pool,
		telemetry: telemetry,
		geocoder:  geolocation.NewReverseGeocoder(),
	}
}

func (s *TelemetryService) SetSSEBroker(broker SSEBroker) {
	s.sse = broker
}

func (s *TelemetryService) SetBikeRepo(repo *repository.BikeRepository) {
	s.bikes = repo
}

func (s *TelemetryService) SetNotificationService(ns *notification.Service) {
	s.notifications = ns
}

// IngestTelemetry reads top to bottom. Every `if` body is a single level deep,
// and no block ever contains another block.
func (s *TelemetryService) IngestTelemetry(ctx context.Context, payload domain.TelemetryPayload) error {
	sensorID := strings.TrimSpace(payload.SensorID)
	if sensorID == "" {
		return ErrInvalidSensorID
	}

	bikeID, geofenceID, err := s.telemetry.ResolveBikeBySensor(ctx, sensorID)
	if err != nil {
		return err
	}

	recordedAt := time.Now().UTC()
	if payload.Timestamp > 0 {
		recordedAt = time.Unix(payload.Timestamp, 0).UTC()
	}

	// Normalize what the device reported.
	speed := math.Max(payload.SpeedKMH, 0)
	heading := payload.HeadingDeg
	switch {
	case heading < 0:
		heading = 0
	case heading > maxHeadingDeg && heading <= maxScaledHeadingDeg:
		heading /= 10.0
	case heading > maxHeadingDeg:
		heading = math.Mod(heading, maxHeadingDeg)
	}

	// Fill in speed and heading from the previous fix when the device omitted them.
	prevLat, prevLng, prevAt, hasPrev, err := s.telemetry.GetBikeLastPosition(ctx, bikeID)
	if err != nil {
		slog.Warn("last position lookup failed", "bike_id", bikeID, "err", err)
		hasPrev = false
	}

	bothFixesValid := hasPrev &&
		prevLat != 0 && prevLng != 0 &&
		payload.Latitude != 0 && payload.Longitude != 0

	distanceKM := 0.0
	if bothFixesValid {
		distanceKM = haversineKM(prevLat, prevLng, payload.Latitude, payload.Longitude)
	}

	hours := recordedAt.Sub(prevAt).Hours()
	hasMoved := bothFixesValid && distanceKM > minTrackedDistanceKM
	canDeriveSpeed := hasMoved && speed <= 0 && hours > 0 && distanceKM/hours < maxPlausibleSpeedKMH
	canDeriveHeading := hasMoved && heading <= 0

	if canDeriveSpeed {
		speed = distanceKM / hours
	}
	if canDeriveHeading {
		heading = bearingDeg(prevLat, prevLng, payload.Latitude, payload.Longitude)
	}

	ignition := payload.Ignition || speed > movingThresholdKMH
	status := "STOPPED"
	if ignition {
		status = "RUNNING"
	}

	battery := payload.BatteryPercent
	if battery < 0 {
		battery = 0
	}
	if battery > 100 {
		battery = 100
	}
	signal := payload.SignalQuality
	if signal < 0 {
		signal = 0
	}

	breached, _, err := s.telemetry.CheckGeofenceBreach(ctx, repository.CheckGeofenceParams{
		GeofenceID: geofenceID,
		Latitude:   payload.Latitude,
		Longitude:  payload.Longitude,
	})
	if err != nil {
		return err
	}

	// Network call happens before the transaction, so no DB connection is held while waiting.
	var address, subCounty, county string
	hasCoordinates := payload.Latitude != 0 || payload.Longitude != 0
	if hasCoordinates {
		geoCtx, cancel := context.WithTimeout(ctx, reverseGeocodeTimeout)
		defer cancel()
		info := s.geocoder.ReverseGeocode(geoCtx, payload.Latitude, payload.Longitude)
		address, subCounty, county = info.FormattedAddress, info.SubCounty, info.County
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	locType := payload.LocType
	if locType == "" {
		locType = "gps"
	}
	accuracy := payload.Accuracy
	if accuracy <= 0 && payload.Latitude != 0 && payload.Longitude != 0 {
		accuracy = 5.0
	}

	err = s.telemetry.SaveTelemetry(ctx, tx, repository.SaveTelemetryParams{
		BikeID:        bikeID,
		SensorID:      sensorID,
		Latitude:      payload.Latitude,
		Longitude:     payload.Longitude,
		Accuracy:      accuracy,
		LocType:       locType,
		Address:       address,
		SubCounty:     subCounty,
		County:        county,
		SpeedKMH:      speed,
		HeadingDeg:    heading,
		Status:        status,
		Ignition:      ignition,
		Battery:       battery,
		SignalQuality: signal,
		Breach:        breached,
		RecordedAt:    recordedAt,
	})
	if err != nil {
		return err
	}
	err = tx.Commit(ctx)
	if err != nil {
		return err
	}

	// Everything below happens after the data is safely stored.
	eventType := strings.TrimSpace(payload.EventType)
	if eventType == "" {
		eventType = strings.TrimSpace(payload.Event)
	}

	var saveEventErr error
	if eventType != "" {
		saveEventErr = s.telemetry.SaveEvent(ctx, repository.SaveEventParams{
			BikeID:     bikeID,
			EventType:  eventType,
			Latitude:   payload.Latitude,
			Longitude:  payload.Longitude,
			RecordedAt: recordedAt,
		})
	}
	if saveEventErr != nil {
		slog.Error("saving event failed", "bike_id", bikeID, "event", eventType, "err", saveEventErr)
	}

	if s.bikes == nil {
		return nil
	}
	bike, err := s.bikes.GetByIDDirect(ctx, bikeID)
	if err != nil || bike == nil {
		slog.Warn("bike lookup for publishing failed", "bike_id", bikeID, "err", err)
		return nil
	}
	dto := ToBikeDTO(bike)
	if s.sse != nil {
		s.sse.BroadcastBikeUpdate(dto.UserID, dto)
	}

	alert := securityEvent{
		bikeID:       bikeID,
		userID:       dto.UserID,
		registration: dto.RegistrationNumber,
		lat:          payload.Latitude,
		lng:          payload.Longitude,
		at:           recordedAt,
	}

	alert.eventType = eventGeofenceBreach
	if breached {
		s.raiseAlert(ctx, alert)
	}

	alert.eventType = eventType
	if eventType != "" && eventType != eventGeofenceBreach {
		s.raiseAlert(ctx, alert)
	}
	return nil
}

func (s *TelemetryService) IngestEvent(
	ctx context.Context,
	sensorID, eventType string,
	lat, lng float64,
	recordedAt time.Time,
) error {
	sensorID = strings.TrimSpace(sensorID)
	if sensorID == "" {
		return ErrInvalidSensorID
	}

	bikeID, _, err := s.telemetry.ResolveBikeBySensor(ctx, sensorID)
	if err != nil {
		return err
	}
	if recordedAt.IsZero() {
		recordedAt = time.Now().UTC()
	}

	err = s.telemetry.SaveEvent(ctx, repository.SaveEventParams{
		BikeID:     bikeID,
		EventType:  eventType,
		Latitude:   lat,
		Longitude:  lng,
		RecordedAt: recordedAt,
	})
	if err != nil {
		return err
	}

	if s.bikes == nil {
		return nil
	}
	bike, err := s.bikes.GetByIDDirect(ctx, bikeID)
	if err != nil || bike == nil || bike.UserID == "" {
		slog.Warn("cannot resolve bike owner; alert skipped", "bike_id", bikeID, "err", err)
		return nil
	}
	registration := bike.RegistrationNumber
	if registration == "" {
		registration = bikeID
	}

	s.raiseAlert(ctx, securityEvent{
		bikeID:       bikeID,
		userID:       bike.UserID,
		registration: registration,
		eventType:    eventType,
		lat:          lat,
		lng:          lng,
		at:           recordedAt,
	})
	return nil
}

// raiseAlert is the single path through which every alert leaves the service.
func (s *TelemetryService) raiseAlert(ctx context.Context, ev securityEvent) {
	title := "🚨 " + strings.ReplaceAll(ev.eventType, "_", " ")
	body := "Security event " + ev.eventType + " detected on " + ev.registration
	if ev.eventType == eventGeofenceBreach {
		title := "🚨 Geofence Breach Alert"
		body := "Motorcycle " + ev.registration + " has breached its geofence boundary."
	}

	if s.sse != nil {
		s.sse.BroadcastAlert(ev.userID, map[string]any{
			"bike_id":     ev.bikeID,
			"event_type":  ev.eventType,
			"title":       title,
			"activity":    body,
			"latitude":    ev.lat,
			"longitude":   ev.lng,
			"recorded_at": ev.at,
		})
	}

	if s.notifications != nil && ev.userID != "" {
		err := s.notifications.DispatchSecurityAlert(ctx, ev.userID, notification.SecurityAlert{
			BikeID:     ev.bikeID,
			Title:      title,
			Body:       body,
			EventType:  ev.eventType,
			Latitude:   ev.lat,
			Longitude:  ev.lng,
			RecordedAt: ev.at,
		})
		if err != nil {
			slog.Error("dispatching alert failed", "bike_id", ev.bikeID, "event", ev.eventType, "err", err)
		}
	}
}

func haversineKM(lat1, lng1, lat2, lng2 float64) float64 {
	const rad = math.Pi / 180.0
	dLat := (lat2 - lat1) * rad
	dLng := (lng2 - lng1) * rad
	h := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*rad)*math.Cos(lat2*rad)*math.Sin(dLng/2)*math.Sin(dLng/2)
	return earthRadiusKM * 2 * math.Atan2(math.Sqrt(h), math.Sqrt(1-h))
}

func bearingDeg(lat1, lng1, lat2, lng2 float64) float64 {
	const rad = math.Pi / 180.0
	phi1, phi2 := lat1*rad, lat2*rad
	dLambda := (lng2 - lng1) * rad
	y := math.Sin(dLambda) * math.Cos(phi2)
	x := math.Cos(phi1)*math.Sin(phi2) - math.Sin(phi1)*math.Cos(phi2)*math.Cos(dLambda)
	return math.Mod(math.Atan2(y, x)*(180.0/math.Pi)+360.0, 360.0)
}
