package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TelemetryRepository struct {
	pool *pgxpool.Pool
}

func NewTelemetryRepository(pool *pgxpool.Pool) *TelemetryRepository {
	return &TelemetryRepository{pool: pool}
}

func (r *TelemetryRepository) ResolveBikeBySensor(ctx context.Context, sensorID string) (string, string, error) {
	queryAssignment := `
		SELECT 
			ta.bike_id, 
			COALESCE(b.geofence_id, '')
		FROM trackers t
		JOIN tracker_assignments ta ON ta.tracker_id = t.id AND ta.unassigned_at IS NULL
		JOIN bikes b ON b.id = ta.bike_id
		WHERE t.imei = $1
		LIMIT 1
	`
	var bikeID, geofenceID string
	err := r.pool.QueryRow(ctx, queryAssignment, sensorID).Scan(&bikeID, &geofenceID)
	if err == nil {
		return bikeID, geofenceID, nil
	}

	queryDirectBike := `
		SELECT 
			b.id, 
			COALESCE(b.geofence_id, '')
		FROM bikes b
		WHERE b.imei = $1
		LIMIT 1
	`
	err = r.pool.QueryRow(ctx, queryDirectBike, sensorID).Scan(&bikeID, &geofenceID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", ErrNotFound
		}
		return "", "", err
	}

	return bikeID, geofenceID, nil
}

func (r *TelemetryRepository) GetBikeLastPosition(ctx context.Context, bikeID string) (float64, float64, time.Time, bool, error) {
	query := `
		SELECT latitude, longitude, last_telemetry_at
		FROM bikes
		WHERE id = $1
		LIMIT 1
	`
	var lat, lng float64
	var lastTime *time.Time
	err := r.pool.QueryRow(ctx, query, bikeID).Scan(&lat, &lng, &lastTime)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, 0, time.Time{}, false, ErrNotFound
		}
		return 0, 0, time.Time{}, false, err
	}

	if lastTime == nil {
		return lat, lng, time.Time{}, false, nil
	}

	return lat, lng, *lastTime, true, nil
}

func (r *TelemetryRepository) CheckGeofenceBreach(ctx context.Context, geofenceID string, lat float64, lng float64) (bool, string, error) {
	if geofenceID == "" {
		return false, "", nil
	}

	query := `
		SELECT 
			name,
			alert_on_exit,
			alert_on_entry,
			boundary IS NOT NULL AND NOT ST_Contains(boundary, ST_SetSRID(ST_Point($1, $2), 4326)) AS is_outside
		FROM geofences
		WHERE id = $3 AND status = 'ACTIVE' AND deleted_at IS NULL
		LIMIT 1
	`
	var name string
	var alertOnExit, alertOnEntry, isOutside bool

	err := r.pool.QueryRow(ctx, query, lng, lat, geofenceID).Scan(&name, &alertOnExit, &alertOnEntry, &isOutside)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, "", nil
		}
		return false, "", err
	}

	breach := false
	if alertOnExit && isOutside {
		breach = true
	}

	return breach, name, nil
}

func (r *TelemetryRepository) SaveTelemetry(
	ctx context.Context,
	tx pgx.Tx,
	bikeID string,
	sensorID string,
	lat float64,
	lng float64,
	speed float64,
	heading float64,
	status string,
	ignition bool,
	battery int,
	signalQuality int,
	breach bool,
	recordedAt time.Time,
) error {
	updateTrackerQuery := `
		UPDATE trackers
		SET last_online_at = $1, updated_at = $1
		WHERE imei = $2
	`
	_, _ = tx.Exec(ctx, updateTrackerQuery, recordedAt, sensorID)

	updateBikeQuery := `
		UPDATE bikes
		SET
			latitude = $1,
			longitude = $2,
			speed_kmh = $3,
			heading_deg = $4,
			status = $5,
			ignition = $6,
			battery_percent = $7,
			signal_quality_csq = $8,
			last_telemetry_at = $9,
			updated_at = $9
		WHERE id = $10
	`
	_, err := tx.Exec(ctx, updateBikeQuery,
		lat, lng, speed, heading, status, ignition, battery, signalQuality, recordedAt, bikeID,
	)
	if err != nil {
		return err
	}

	insertBreadcrumbQuery := `
		INSERT INTO bike_breadcrumbs (
			bike_id, latitude, longitude, heading_deg, speed_kmh, ignition, battery_percent, geofence_breach, recorded_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		)
	`
	_, err = tx.Exec(ctx, insertBreadcrumbQuery,
		bikeID, lat, lng, heading, speed, ignition, battery, breach, recordedAt,
	)
	return err
}

func (r *TelemetryRepository) GetGeofenceIDForBike(ctx context.Context, bikeID string) (sql.NullString, error) {
	query := `SELECT geofence_id FROM bikes WHERE id = $1 LIMIT 1`
	var geofenceID sql.NullString
	err := r.pool.QueryRow(ctx, query, bikeID).Scan(&geofenceID)
	return geofenceID, err
}

func (r *TelemetryRepository) SaveEvent(
	ctx context.Context,
	bikeID string,
	eventType string,
	lat float64,
	lng float64,
	recordedAt time.Time,
) error {
	query := `
		INSERT INTO bike_events (bike_id, event_type, latitude, longitude, recorded_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.pool.Exec(ctx, query, bikeID, eventType, lat, lng, recordedAt)
	return err
}
