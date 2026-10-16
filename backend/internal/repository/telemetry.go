package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// --- Parameter Types ---

type CheckGeofenceParams struct {
	GeofenceID string
	Latitude   float64
	Longitude  float64
}

type SaveTelemetryParams struct {
	BikeID        string
	SensorID      string
	Latitude      float64
	Longitude     float64
	Accuracy      float64
	LocType       string
	Address       string
	SubCounty     string
	County        string
	SpeedKMH      float64
	HeadingDeg    float64
	Status        string
	Ignition      bool
	Battery       int
	SignalQuality int
	Breach        bool
	RecordedAt    time.Time
}

type SaveEventParams struct {
	BikeID     string
	EventType  string
	Latitude   float64
	Longitude  float64
	RecordedAt time.Time
}

// --- Repository ---

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

func (r *TelemetryRepository) CheckGeofenceBreach(ctx context.Context, params CheckGeofenceParams) (bool, string, error) {
	if params.GeofenceID == "" {
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

	err := r.pool.QueryRow(ctx, query, params.Longitude, params.Latitude, params.GeofenceID).Scan(&name, &alertOnExit, &alertOnEntry, &isOutside)
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
	params SaveTelemetryParams,
) error {
	updateTrackerQuery := `
		UPDATE trackers
		SET last_online_at = $1, updated_at = $1
		WHERE imei = $2
	`
	_, _ = tx.Exec(ctx, updateTrackerQuery, params.RecordedAt, params.SensorID)

	updateBikeQuery := `
		UPDATE bikes
		SET
			latitude = CASE WHEN $1 = 0.0 AND $2 = 0.0 THEN latitude ELSE $1 END,
			longitude = CASE WHEN $1 = 0.0 AND $2 = 0.0 THEN longitude ELSE $2 END,
			accuracy = CASE WHEN $1 = 0.0 AND $2 = 0.0 THEN accuracy ELSE $3 END,
			loc_type = CASE WHEN $1 = 0.0 AND $2 = 0.0 THEN loc_type ELSE $4 END,
			current_address = COALESCE(NULLIF($5, ''), current_address),
			sub_county = COALESCE(NULLIF($6, ''), sub_county),
			county = COALESCE(NULLIF($7, ''), county),
			speed_kmh = $8,
			heading_deg = $9,
			status = $10,
			ignition = $11,
			battery_percent = $12,
			signal_quality_csq = $13,
			last_telemetry_at = $14,
			updated_at = $14
		WHERE id = $15
	`
	_, err := tx.Exec(ctx, updateBikeQuery,
		params.Latitude, params.Longitude, params.Accuracy, params.LocType,
		params.Address, params.SubCounty, params.County,
		params.SpeedKMH, params.HeadingDeg, params.Status, params.Ignition,
		params.Battery, params.SignalQuality, params.RecordedAt, params.BikeID,
	)
	if err != nil {
		return err
	}

	insertBreadcrumbQuery := `
		INSERT INTO bike_breadcrumbs (
			bike_id, latitude, longitude, accuracy, loc_type, heading_deg, speed_kmh, ignition, battery_percent, address, geofence_breach, recorded_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		)
	`
	_, err = tx.Exec(ctx, insertBreadcrumbQuery,
		params.BikeID, params.Latitude, params.Longitude, params.Accuracy, params.LocType,
		params.HeadingDeg, params.SpeedKMH, params.Ignition, params.Battery, params.Address,
		params.Breach, params.RecordedAt,
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
	params SaveEventParams,
) error {
	query := `
		INSERT INTO bike_events (bike_id, event_type, latitude, longitude, recorded_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.pool.Exec(ctx, query, params.BikeID, params.EventType, params.Latitude, params.Longitude, params.RecordedAt)
	return err
}
