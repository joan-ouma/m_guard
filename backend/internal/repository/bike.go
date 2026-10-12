package repository

import (
	"context"
	"database/sql"
	"errors"

	"motoguard-backend/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type BikeRepository struct {
	pool *pgxpool.Pool
}

func NewBikeRepository(pool *pgxpool.Pool) *BikeRepository {
	return &BikeRepository{pool: pool}
}

func (r *BikeRepository) Create(ctx context.Context, tx pgx.Tx, b *domain.Bike) error {
	var geofenceVal any
	if b.GeofenceID != "" {
		geofenceVal = b.GeofenceID
	}
	var driverVal any
	if b.DriverContactID != "" {
		driverVal = b.DriverContactID
	}
<<<<<<< HEAD
=======
	var riderNameVal any
	if b.RiderName != "" {
		riderNameVal = b.RiderName
	}
	var riderIDVal any
	if b.RiderNationalID != "" {
		riderIDVal = b.RiderNationalID
	}
	var riderPhoneVal any
	if b.RiderPhone != "" {
		riderPhoneVal = b.RiderPhone
	}
>>>>>>> origin/main

	query := `
		INSERT INTO bikes (
			id, user_id, registration_number, label, make, model, year, color, imei,
			status, ignition, speed_kmh, heading_deg, latitude, longitude, current_address, sub_county, county,
<<<<<<< HEAD
			battery_percent, signal_quality_csq, last_telemetry_at, geofence_id, driver_contact_id, is_immobilized, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9,
			$10, $11, $12, $13, $14, $15, $16, $17, $18,
			$19, $20, $21, $22, $23, $24, $25, $26
=======
			battery_percent, signal_quality_csq, last_telemetry_at, geofence_id, driver_contact_id,
			rider_name, rider_national_id, rider_phone,
			is_immobilized, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9,
			$10, $11, $12, $13, $14, $15, $16, $17, $18,
			$19, $20, $21, $22, $23,
			$24, $25, $26,
			$27, $28, $29
>>>>>>> origin/main
		)
	`
	_, err := tx.Exec(ctx, query,
		b.ID, b.UserID, b.RegistrationNumber, b.Label, b.Make, b.Model, b.Year, b.Color, b.IMEI,
		b.Status, b.Ignition, b.SpeedKMH, b.HeadingDeg, b.Latitude, b.Longitude, b.CurrentAddress, b.SubCounty, b.County,
<<<<<<< HEAD
		b.BatteryPercent, b.SignalQualityCSQ, b.LastTelemetryAt, geofenceVal, driverVal, b.IsImmobilized, b.CreatedAt, b.UpdatedAt,
=======
		b.BatteryPercent, b.SignalQualityCSQ, b.LastTelemetryAt, geofenceVal, driverVal,
		riderNameVal, riderIDVal, riderPhoneVal,
		b.IsImmobilized, b.CreatedAt, b.UpdatedAt,
>>>>>>> origin/main
	)
	return err
}

func (r *BikeRepository) GetByID(ctx context.Context, userID string, bikeID string) (*domain.Bike, error) {
	query := `
		SELECT 
			id, user_id, registration_number, label, make, model, year, color, imei,
			status, ignition, speed_kmh, heading_deg, latitude, longitude, current_address, sub_county, county,
<<<<<<< HEAD
			battery_percent, signal_quality_csq, last_telemetry_at, geofence_id, driver_contact_id, is_immobilized, created_at, updated_at
=======
			battery_percent, signal_quality_csq, last_telemetry_at, geofence_id, driver_contact_id,
			rider_name, rider_national_id, rider_phone,
			is_immobilized, created_at, updated_at
>>>>>>> origin/main
		FROM bikes
		WHERE user_id = $1 AND id = $2
		LIMIT 1
	`
	row := r.pool.QueryRow(ctx, query, userID, bikeID)

	var b domain.Bike
<<<<<<< HEAD
	var geofenceVal, driverVal, addrVal, subCountyVal, countyVal sql.NullString

	err := row.Scan(
		&b.ID, &b.UserID, &b.RegistrationNumber, &b.Label, &b.Make, &b.Model, &b.Year, &b.Color, &b.IMEI,
		&b.Status, &b.Ignition, &b.SpeedKMH, &b.HeadingDeg, &b.Latitude, &b.Longitude, &addrVal, &subCountyVal, &countyVal,
		&b.BatteryPercent, &b.SignalQualityCSQ, &b.LastTelemetryAt, &geofenceVal, &driverVal, &b.IsImmobilized, &b.CreatedAt, &b.UpdatedAt,
=======
	var geofenceVal, driverVal, riderNameVal, riderIDVal, riderPhoneVal sql.NullString

	err := row.Scan(
		&b.ID, &b.UserID, &b.RegistrationNumber, &b.Label, &b.Make, &b.Model, &b.Year, &b.Color, &b.IMEI,
		&b.Status, &b.Ignition, &b.SpeedKMH, &b.HeadingDeg, &b.Latitude, &b.Longitude, &b.CurrentAddress, &b.SubCounty, &b.County,
		&b.BatteryPercent, &b.SignalQualityCSQ, &b.LastTelemetryAt, &geofenceVal, &driverVal,
		&riderNameVal, &riderIDVal, &riderPhoneVal,
		&b.IsImmobilized, &b.CreatedAt, &b.UpdatedAt,
>>>>>>> origin/main
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if geofenceVal.Valid {
		b.GeofenceID = geofenceVal.String
	}
	if driverVal.Valid {
		b.DriverContactID = driverVal.String
	}
<<<<<<< HEAD
	if addrVal.Valid {
		b.CurrentAddress = addrVal.String
	}
	if subCountyVal.Valid {
		b.SubCounty = subCountyVal.String
	}
	if countyVal.Valid {
		b.County = countyVal.String
=======
	if riderNameVal.Valid {
		b.RiderName = riderNameVal.String
	}
	if riderIDVal.Valid {
		b.RiderNationalID = riderIDVal.String
	}
	if riderPhoneVal.Valid {
		b.RiderPhone = riderPhoneVal.String
>>>>>>> origin/main
	}

	return &b, nil
}

func (r *BikeRepository) ListByOwner(ctx context.Context, userID string) ([]domain.Bike, error) {
	query := `
		SELECT 
			id, user_id, registration_number, label, make, model, year, color, imei,
			status, ignition, speed_kmh, heading_deg, latitude, longitude, current_address, sub_county, county,
<<<<<<< HEAD
			battery_percent, signal_quality_csq, last_telemetry_at, geofence_id, driver_contact_id, is_immobilized, created_at, updated_at
=======
			battery_percent, signal_quality_csq, last_telemetry_at, geofence_id, driver_contact_id,
			rider_name, rider_national_id, rider_phone,
			is_immobilized, created_at, updated_at
>>>>>>> origin/main
		FROM bikes
		WHERE user_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bikes []domain.Bike
	for rows.Next() {
		var b domain.Bike
<<<<<<< HEAD
		var geofenceVal, driverVal, addrVal, subCountyVal, countyVal sql.NullString
		err := rows.Scan(
			&b.ID, &b.UserID, &b.RegistrationNumber, &b.Label, &b.Make, &b.Model, &b.Year, &b.Color, &b.IMEI,
			&b.Status, &b.Ignition, &b.SpeedKMH, &b.HeadingDeg, &b.Latitude, &b.Longitude, &addrVal, &subCountyVal, &countyVal,
			&b.BatteryPercent, &b.SignalQualityCSQ, &b.LastTelemetryAt, &geofenceVal, &driverVal, &b.IsImmobilized, &b.CreatedAt, &b.UpdatedAt,
=======
		var geofenceVal, driverVal, riderNameVal, riderIDVal, riderPhoneVal sql.NullString
		err := rows.Scan(
			&b.ID, &b.UserID, &b.RegistrationNumber, &b.Label, &b.Make, &b.Model, &b.Year, &b.Color, &b.IMEI,
			&b.Status, &b.Ignition, &b.SpeedKMH, &b.HeadingDeg, &b.Latitude, &b.Longitude, &b.CurrentAddress, &b.SubCounty, &b.County,
			&b.BatteryPercent, &b.SignalQualityCSQ, &b.LastTelemetryAt, &geofenceVal, &driverVal,
			&riderNameVal, &riderIDVal, &riderPhoneVal,
			&b.IsImmobilized, &b.CreatedAt, &b.UpdatedAt,
>>>>>>> origin/main
		)
		if err != nil {
			return nil, err
		}
		if geofenceVal.Valid {
			b.GeofenceID = geofenceVal.String
		}
		if driverVal.Valid {
			b.DriverContactID = driverVal.String
		}
<<<<<<< HEAD
		if addrVal.Valid {
			b.CurrentAddress = addrVal.String
		}
		if subCountyVal.Valid {
			b.SubCounty = subCountyVal.String
		}
		if countyVal.Valid {
			b.County = countyVal.String
=======
		if riderNameVal.Valid {
			b.RiderName = riderNameVal.String
		}
		if riderIDVal.Valid {
			b.RiderNationalID = riderIDVal.String
		}
		if riderPhoneVal.Valid {
			b.RiderPhone = riderPhoneVal.String
>>>>>>> origin/main
		}
		bikes = append(bikes, b)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return bikes, nil
}

func (r *BikeRepository) SetImmobilized(ctx context.Context, userID string, bikeID string, isImmobilized bool) (string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	var currentImmobilized bool
	var imei string
	var lat, lng float64
	checkQuery := `
		SELECT is_immobilized, imei, latitude, longitude
		FROM bikes
		WHERE user_id = $1 AND id = $2
	`
	err = tx.QueryRow(ctx, checkQuery, userID, bikeID).Scan(&currentImmobilized, &imei, &lat, &lng)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", err
	}

	// State is already at desired value; avoid recording duplicate events
	if currentImmobilized == isImmobilized {
		return imei, nil
	}

	updateQuery := `
		UPDATE bikes
		SET is_immobilized = $1, updated_at = NOW()
		WHERE user_id = $2 AND id = $3
	`
	if _, err := tx.Exec(ctx, updateQuery, isImmobilized, userID, bikeID); err != nil {
		return "", err
	}

	eventType := "IMMOBILIZED"
	if !isImmobilized {
		eventType = "REACTIVATED"
	}

	eventQuery := `
		INSERT INTO bike_events (bike_id, event_type, latitude, longitude, recorded_at)
		VALUES ($1, $2, $3, $4, NOW())
	`
	if _, err := tx.Exec(ctx, eventQuery, bikeID, eventType, lat, lng); err != nil {
		return "", err
	}

	if err := tx.Commit(ctx); err != nil {
		return "", err
	}

	return imei, nil
}

func (r *BikeRepository) Update(ctx context.Context, tx pgx.Tx, b *domain.Bike) error {
	var geofenceVal any
	if b.GeofenceID != "" {
		geofenceVal = b.GeofenceID
	}
	var driverVal any
	if b.DriverContactID != "" {
		driverVal = b.DriverContactID
	}
<<<<<<< HEAD
=======
	var riderNameVal any
	if b.RiderName != "" {
		riderNameVal = b.RiderName
	}
	var riderIDVal any
	if b.RiderNationalID != "" {
		riderIDVal = b.RiderNationalID
	}
	var riderPhoneVal any
	if b.RiderPhone != "" {
		riderPhoneVal = b.RiderPhone
	}
>>>>>>> origin/main

	query := `
		UPDATE bikes
		SET
			registration_number = $1,
			label = $2,
			make = $3,
			model = $4,
			year = $5,
			color = $6,
			status = $7,
			current_address = $8,
			sub_county = $9,
			county = $10,
			geofence_id = $11,
			driver_contact_id = $12,
<<<<<<< HEAD
			updated_at = $13
		WHERE user_id = $14 AND id = $15
=======
			rider_name = $13,
			rider_national_id = $14,
			rider_phone = $15,
			updated_at = $16
		WHERE user_id = $17 AND id = $18
>>>>>>> origin/main
	`
	tag, err := tx.Exec(ctx, query,
		b.RegistrationNumber, b.Label, b.Make, b.Model, b.Year, b.Color,
		b.Status, b.CurrentAddress, b.SubCounty, b.County,
<<<<<<< HEAD
		geofenceVal, driverVal, b.UpdatedAt,
=======
		geofenceVal, driverVal,
		riderNameVal, riderIDVal, riderPhoneVal,
		b.UpdatedAt,
>>>>>>> origin/main
		b.UserID, b.ID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *BikeRepository) Delete(ctx context.Context, tx pgx.Tx, userID string, bikeID string) error {
	query := `
		DELETE FROM bikes
		WHERE user_id = $1 AND id = $2
	`
	tag, err := tx.Exec(ctx, query, userID, bikeID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *BikeRepository) ExistsIMEI(ctx context.Context, imei string, excludeBikeID string) (bool, error) {
<<<<<<< HEAD
	query := `
		SELECT 1 FROM bikes WHERE imei = $1 AND id != $2 LIMIT 1
	`
	row := r.pool.QueryRow(ctx, query, imei, excludeBikeID)
=======
	var row pgx.Row
	if excludeBikeID != "" {
		query := `SELECT 1 FROM bikes WHERE imei = $1 AND id != $2 LIMIT 1`
		row = r.pool.QueryRow(ctx, query, imei, excludeBikeID)
	} else {
		query := `SELECT 1 FROM bikes WHERE imei = $1 LIMIT 1`
		row = r.pool.QueryRow(ctx, query, imei)
	}
>>>>>>> origin/main
	var val int
	err := row.Scan(&val)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *BikeRepository) ExistsRegistrationNumber(ctx context.Context, userID string, regNumber string, excludeBikeID string) (bool, error) {
<<<<<<< HEAD
	query := `
		SELECT 1 FROM bikes WHERE user_id = $1 AND LOWER(registration_number) = LOWER($2) AND id != $3 LIMIT 1
	`
	row := r.pool.QueryRow(ctx, query, userID, regNumber, excludeBikeID)
=======
	var row pgx.Row
	if excludeBikeID != "" {
		query := `SELECT 1 FROM bikes WHERE user_id = $1 AND LOWER(registration_number) = LOWER($2) AND id != $3 LIMIT 1`
		row = r.pool.QueryRow(ctx, query, userID, regNumber, excludeBikeID)
	} else {
		query := `SELECT 1 FROM bikes WHERE user_id = $1 AND LOWER(registration_number) = LOWER($2) LIMIT 1`
		row = r.pool.QueryRow(ctx, query, userID, regNumber)
	}
>>>>>>> origin/main
	var val int
	err := row.Scan(&val)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *BikeRepository) GetBreadcrumbs(ctx context.Context, userID string, bikeID string, startDate string, endDate string, limit int) ([]domain.BreadcrumbDTO, error) {
	if limit <= 0 || limit > 5000 {
		limit = 1000
	}

	var query string
	var args []any

	if startDate != "" && endDate != "" {
		query = `
			SELECT bb.id, bb.bike_id, bb.latitude, bb.longitude, bb.speed_kmh, bb.heading_deg, bb.ignition, bb.battery_percent, bb.address, bb.geofence_breach, bb.recorded_at
			FROM bike_breadcrumbs bb
			JOIN bikes b ON b.id = bb.bike_id
<<<<<<< HEAD
			WHERE b.user_id = $1 AND bb.bike_id = $2 AND bb.recorded_at >= $3 AND bb.recorded_at <= $4 AND bb.latitude != 0 AND bb.longitude != 0
=======
			WHERE b.user_id = $1 AND bb.bike_id = $2 AND bb.recorded_at >= $3 AND bb.recorded_at <= $4
>>>>>>> origin/main
			ORDER BY bb.recorded_at ASC
			LIMIT $5
		`
		args = []any{userID, bikeID, startDate, endDate, limit}
	} else if startDate != "" {
		query = `
			SELECT bb.id, bb.bike_id, bb.latitude, bb.longitude, bb.speed_kmh, bb.heading_deg, bb.ignition, bb.battery_percent, bb.address, bb.geofence_breach, bb.recorded_at
			FROM bike_breadcrumbs bb
			JOIN bikes b ON b.id = bb.bike_id
<<<<<<< HEAD
			WHERE b.user_id = $1 AND bb.bike_id = $2 AND bb.recorded_at >= $3 AND bb.latitude != 0 AND bb.longitude != 0
=======
			WHERE b.user_id = $1 AND bb.bike_id = $2 AND bb.recorded_at >= $3
>>>>>>> origin/main
			ORDER BY bb.recorded_at ASC
			LIMIT $4
		`
		args = []any{userID, bikeID, startDate, limit}
	} else {
		query = `
			SELECT bb.id, bb.bike_id, bb.latitude, bb.longitude, bb.speed_kmh, bb.heading_deg, bb.ignition, bb.battery_percent, bb.address, bb.geofence_breach, bb.recorded_at
			FROM bike_breadcrumbs bb
			JOIN bikes b ON b.id = bb.bike_id
<<<<<<< HEAD
			WHERE b.user_id = $1 AND bb.bike_id = $2 AND bb.latitude != 0 AND bb.longitude != 0
=======
			WHERE b.user_id = $1 AND bb.bike_id = $2
>>>>>>> origin/main
			ORDER BY bb.recorded_at ASC
			LIMIT $3
		`
		args = []any{userID, bikeID, limit}
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []domain.BreadcrumbDTO
	for rows.Next() {
		var p domain.BreadcrumbDTO
		err := rows.Scan(
			&p.ID, &p.BikeID, &p.Latitude, &p.Longitude,
			&p.SpeedKMH, &p.HeadingDeg, &p.Ignition,
			&p.BatteryPercent, &p.Address, &p.GeofenceBreach, &p.RecordedAt,
		)
		if err != nil {
			return nil, err
		}
		points = append(points, p)
	}

	return points, rows.Err()
}

func (r *BikeRepository) GetEvents(ctx context.Context, userID string, bikeID string, startDate string, endDate string, limit int) ([]domain.BikeEventDTO, error) {
	if limit <= 0 || limit > 5000 {
		limit = 1000
	}

	var query string
	var args []any

	if startDate != "" && endDate != "" {
		query = `
			SELECT be.id, be.bike_id, be.event_type, be.latitude, be.longitude, be.recorded_at
			FROM bike_events be
			JOIN bikes b ON b.id = be.bike_id
<<<<<<< HEAD
			WHERE b.user_id = $1 AND be.bike_id = $2 AND be.recorded_at >= $3 AND be.recorded_at <= $4 AND be.latitude != 0 AND be.longitude != 0
=======
			WHERE b.user_id = $1 AND be.bike_id = $2 AND be.recorded_at >= $3 AND be.recorded_at <= $4
>>>>>>> origin/main
			ORDER BY be.recorded_at ASC
			LIMIT $5
		`
		args = []any{userID, bikeID, startDate, endDate, limit}
	} else if startDate != "" {
		query = `
			SELECT be.id, be.bike_id, be.event_type, be.latitude, be.longitude, be.recorded_at
			FROM bike_events be
			JOIN bikes b ON b.id = be.bike_id
<<<<<<< HEAD
			WHERE b.user_id = $1 AND be.bike_id = $2 AND be.recorded_at >= $3 AND be.latitude != 0 AND be.longitude != 0
=======
			WHERE b.user_id = $1 AND be.bike_id = $2 AND be.recorded_at >= $3
>>>>>>> origin/main
			ORDER BY be.recorded_at ASC
			LIMIT $4
		`
		args = []any{userID, bikeID, startDate, limit}
	} else {
		query = `
			SELECT be.id, be.bike_id, be.event_type, be.latitude, be.longitude, be.recorded_at
			FROM bike_events be
			JOIN bikes b ON b.id = be.bike_id
<<<<<<< HEAD
			WHERE b.user_id = $1 AND be.bike_id = $2 AND be.latitude != 0 AND be.longitude != 0
=======
			WHERE b.user_id = $1 AND be.bike_id = $2
>>>>>>> origin/main
			ORDER BY be.recorded_at ASC
			LIMIT $3
		`
		args = []any{userID, bikeID, limit}
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []domain.BikeEventDTO
	for rows.Next() {
		var e domain.BikeEventDTO
		err := rows.Scan(
			&e.ID, &e.BikeID, &e.EventType, &e.Latitude, &e.Longitude, &e.RecordedAt,
		)
		if err != nil {
			return nil, err
		}
		events = append(events, e)
	}

	return events, rows.Err()
}
