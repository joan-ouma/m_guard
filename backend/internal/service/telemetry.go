package service

import (
	"context"
	"math"
	"strings"
	"time"

	"motoguard-backend/internal/domain"
	"motoguard-backend/internal/geolocation"
	"motoguard-backend/internal/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

type TelemetryService struct {
	pool            *pgxpool.Pool
	repo            *repository.TelemetryRepository
	bikeRepo        *repository.BikeRepository
	sseBroker       SSEBroker
	reverseGeocoder *geolocation.ReverseGeocoder
}

func NewTelemetryService(pool *pgxpool.Pool, repo *repository.TelemetryRepository) *TelemetryService {
	return &TelemetryService{
		pool:            pool,
		repo:            repo,
		reverseGeocoder: geolocation.NewReverseGeocoder(),
	}
}

func (s *TelemetryService) SetSSEBroker(broker SSEBroker) {
	s.sseBroker = broker
}

func (s *TelemetryService) SetBikeRepo(repo *repository.BikeRepository) {
	s.bikeRepo = repo
}

func (s *TelemetryService) IngestTelemetry(ctx context.Context, payload domain.TelemetryPayload) error {
	sensorID := strings.TrimSpace(payload.SensorID)
	if sensorID == "" {
		return repository.ErrNotFound
	}

	bikeID, geofenceID, err := s.repo.ResolveBikeBySensor(ctx, sensorID)
	if err != nil {
		return err
	}

	recordedAt := time.Now().UTC()
	if payload.Timestamp > 0 {
		recordedAt = time.Unix(payload.Timestamp, 0).UTC()
	}

	prevLat, prevLng, prevTime, hasPrev, err := s.repo.GetBikeLastPosition(ctx, bikeID)
	if err != nil {
		hasPrev = false
	}

	speed := payload.SpeedKMH
	if speed < 0.0 {
		speed = 0.0
	}

	heading := payload.HeadingDeg
	if heading > 360.0 && heading <= 3600.0 {
		heading = heading / 10.0
	} else if heading > 360.0 {
		heading = math.Mod(heading, 360.0)
	}
	if heading < 0.0 {
		heading = 0.0
	}

	if hasPrev && payload.Latitude != 0 && payload.Longitude != 0 && prevLat != 0 && prevLng != 0 {
		distanceKM := haversineDistanceKM(prevLat, prevLng, payload.Latitude, payload.Longitude)
		elapsedHours := recordedAt.Sub(prevTime).Hours()

		if speed <= 0 && elapsedHours > 0 && distanceKM > 0.005 {
			calculatedSpeed := distanceKM / elapsedHours
			if calculatedSpeed < 180.0 {
				speed = calculatedSpeed
			}
		}

		if heading <= 0 && distanceKM > 0.005 {
			heading = calculateBearingDeg(prevLat, prevLng, payload.Latitude, payload.Longitude)
		}
	}

	status := "STOPPED"
	ignition := payload.Ignition
	if ignition || speed > 1.0 || payload.SpeedKMH > 1.0 {
		status = "RUNNING"
	}
	if speed > 1.0 || payload.SpeedKMH > 1.0 {
		ignition = true
	}

	battery := payload.BatteryPercent
	if battery <= 0 {
		battery = 100
	}

	signalQuality := payload.SignalQuality
	if signalQuality <= 0 {
		signalQuality = 20
	}

	breach, _, _ := s.repo.CheckGeofenceBreach(ctx, geofenceID, payload.Latitude, payload.Longitude)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var address, subCounty, county string
	if payload.Latitude != 0 || payload.Longitude != 0 {
		geoCtx, geoCancel := context.WithTimeout(ctx, 4*time.Second)
		addrInfo := s.reverseGeocoder.ReverseGeocode(geoCtx, payload.Latitude, payload.Longitude)
		geoCancel()
		address = addrInfo.FormattedAddress
		subCounty = addrInfo.SubCounty
		county = addrInfo.County
	}

	err = s.repo.SaveTelemetry(
		ctx,
		tx,
		bikeID,
		sensorID,
		payload.Latitude,
		payload.Longitude,
		address,
		subCounty,
		county,
		speed,
		heading,
		status,
		ignition,
		battery,
		signalQuality,
		breach,
		recordedAt,
	)
	if err != nil {
		return err
	}

	evType := strings.TrimSpace(payload.EventType)
	if evType == "" {
		evType = strings.TrimSpace(payload.Event)
	}
	if evType != "" {
		_ = s.repo.SaveEvent(ctx, bikeID, evType, payload.Latitude, payload.Longitude, recordedAt)
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	if s.sseBroker != nil && s.bikeRepo != nil {
		if updatedBike, err := s.bikeRepo.GetByIDDirect(ctx, bikeID); err == nil && updatedBike != nil {
			dto := ToBikeDTO(updatedBike)
			s.sseBroker.BroadcastBikeUpdate(dto.UserID, dto)
		}
	}

	return nil
}

func (s *TelemetryService) IngestEvent(ctx context.Context, sensorID string, eventType string, lat float64, lng float64, recordedAt time.Time) error {
	sensorID = strings.TrimSpace(sensorID)
	if sensorID == "" {
		return repository.ErrNotFound
	}
	bikeID, _, err := s.repo.ResolveBikeBySensor(ctx, sensorID)
	if err != nil {
		return err
	}
	if recordedAt.IsZero() {
		recordedAt = time.Now().UTC()
	}

	err = s.repo.SaveEvent(ctx, bikeID, eventType, lat, lng, recordedAt)
	if err == nil && s.sseBroker != nil {
		s.sseBroker.BroadcastAlert("", map[string]any{
			"bike_id":     bikeID,
			"event_type":  eventType,
			"latitude":    lat,
			"longitude":   lng,
			"recorded_at": recordedAt,
		})
	}
	return err
}

func haversineDistanceKM(lat1, lon1, lat2, lon2 float64) float64 {
	rad := math.Pi / 180.0
	dLat := (lat2 - lat1) * rad
	dLon := (lon2 - lon1) * rad

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*rad)*math.Cos(lat2*rad)*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return 6371.0 * c
}

func calculateBearingDeg(lat1, lon1, lat2, lon2 float64) float64 {
	rad := math.Pi / 180.0
	phi1 := lat1 * rad
	phi2 := lat2 * rad
	deltaLambda := (lon2 - lon1) * rad

	y := math.Sin(deltaLambda) * math.Cos(phi2)
	x := math.Cos(phi1)*math.Sin(phi2) - math.Sin(phi1)*math.Cos(phi2)*math.Cos(deltaLambda)
	theta := math.Atan2(y, x) * (180.0 / math.Pi)

	return math.Mod(theta+360.0, 360.0)
}
