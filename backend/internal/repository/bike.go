package repository

import (
	"context"
	"errors"
	"fmt"
	"strconv"

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

	query := `
		INSERT INTO bikes (
			id, user_id, registration_number, label, make, model, year, color, imei,
			status, ignition, speed_kmh, heading_deg, latitude, longitude, current_address, sub_county, county,
			battery_percent, signal_quality_csq, last_telemetry_at, geofence_id, driver_contact_id,
			is_immobilized, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9,
			$10, $11, $12, $13, $14, $15, $16, $17, $18,
			$19, $20, $21, $22, $23,
			$24, $25, $26
		)
	`
	_, err := tx.Exec(ctx, query,
		b.ID, b.UserID, b.RegistrationNumber, b.Label, b.Make, b.Model, b.Year, b.Color, b.IMEI,
		b.Status, b.Ignition, b.SpeedKMH, b.HeadingDeg, b.Latitude, b.Longitude, b.CurrentAddress, b.SubCounty, b.County,
		b.BatteryPercent, b.SignalQualityCSQ, b.LastTelemetryAt, geofenceVal, driverVal,
		b.IsImmobilized, b.CreatedAt, b.UpdatedAt,
	)
	return err
}

func (r *BikeRepository) GetByID(ctx context.Context, userID string, bikeID string) (*domain.Bike, error) {
	query := `
		SELECT 
			id, user_id, registration_number,
			COALESCE(label, ''), COALESCE(make, ''), COALESCE(model, ''), COALESCE(year, 2024), COALESCE(color, ''), imei,
			COALESCE(status, 'STOPPED'), COALESCE(ignition, false), COALESCE(speed_kmh, 0.0), COALESCE(heading_deg, 0.0),
			COALESCE(latitude, 0.0), COALESCE(longitude, 0.0),
			COALESCE(current_address, ''), COALESCE(sub_county, ''), COALESCE(county, ''),
			COALESCE(battery_percent, 100), COALESCE(signal_quality_csq, 20), last_telemetry_at,
			COALESCE(geofence_id, ''), COALESCE(driver_contact_id, ''),
			COALESCE(is_immobilized, false), created_at, updated_at
		FROM bikes
		WHERE user_id = $1 AND id = $2
		LIMIT 1
	`
	row := r.pool.QueryRow(ctx, query, userID, bikeID)

	var b domain.Bike
	err := row.Scan(
		&b.ID, &b.UserID, &b.RegistrationNumber,
		&b.Label, &b.Make, &b.Model, &b.Year, &b.Color, &b.IMEI,
		&b.Status, &b.Ignition, &b.SpeedKMH, &b.HeadingDeg,
		&b.Latitude, &b.Longitude,
		&b.CurrentAddress, &b.SubCounty, &b.County,
		&b.BatteryPercent, &b.SignalQualityCSQ, &b.LastTelemetryAt,
		&b.GeofenceID, &b.DriverContactID,
		&b.IsImmobilized, &b.CreatedAt, &b.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &b, nil
}

func (r *BikeRepository) GetByIDDirect(ctx context.Context, bikeID string) (*domain.Bike, error) {
	query := `
		SELECT 
			id, user_id, registration_number,
			COALESCE(label, ''), COALESCE(make, ''), COALESCE(model, ''), COALESCE(year, 2024), COALESCE(color, ''), imei,
			COALESCE(status, 'STOPPED'), COALESCE(ignition, false), COALESCE(speed_kmh, 0.0), COALESCE(heading_deg, 0.0),
			COALESCE(latitude, 0.0), COALESCE(longitude, 0.0),
			COALESCE(current_address, ''), COALESCE(sub_county, ''), COALESCE(county, ''),
			COALESCE(battery_percent, 100), COALESCE(signal_quality_csq, 20), last_telemetry_at,
			COALESCE(geofence_id, ''), COALESCE(driver_contact_id, ''),
			COALESCE(is_immobilized, false), created_at, updated_at
		FROM bikes
		WHERE id = $1
		LIMIT 1
	`
	row := r.pool.QueryRow(ctx, query, bikeID)

	var b domain.Bike
	err := row.Scan(
		&b.ID, &b.UserID, &b.RegistrationNumber,
		&b.Label, &b.Make, &b.Model, &b.Year, &b.Color, &b.IMEI,
		&b.Status, &b.Ignition, &b.SpeedKMH, &b.HeadingDeg,
		&b.Latitude, &b.Longitude,
		&b.CurrentAddress, &b.SubCounty, &b.County,
		&b.BatteryPercent, &b.SignalQualityCSQ, &b.LastTelemetryAt,
		&b.GeofenceID, &b.DriverContactID,
		&b.IsImmobilized, &b.CreatedAt, &b.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &b, nil
}

func (r *BikeRepository) ListByOwner(ctx context.Context, userID string, cursor string, limit int) ([]domain.Bike, domain.BikeListMeta, error) {
	if limit <= 0 || limit > 100 {
		limit = 25
	}

	var total int
	countQuery := `SELECT COUNT(*) FROM bikes WHERE user_id = $1`
	_ = r.pool.QueryRow(ctx, countQuery, userID).Scan(&total)

	var query string
	var args []any
	if cursor != "" {
		query = `
			SELECT 
				id, user_id, registration_number,
				COALESCE(label, ''), COALESCE(make, ''), COALESCE(model, ''), COALESCE(year, 2024), COALESCE(color, ''), imei,
				COALESCE(status, 'STOPPED'), COALESCE(ignition, false), COALESCE(speed_kmh, 0.0), COALESCE(heading_deg, 0.0),
				COALESCE(latitude, 0.0), COALESCE(longitude, 0.0),
				COALESCE(current_address, ''), COALESCE(sub_county, ''), COALESCE(county, ''),
				COALESCE(battery_percent, 100), COALESCE(signal_quality_csq, 20), last_telemetry_at,
				COALESCE(geofence_id, ''), COALESCE(driver_contact_id, ''),
				COALESCE(is_immobilized, false), created_at, updated_at
			FROM bikes
			WHERE user_id = $1 AND (
				created_at < (SELECT created_at FROM bikes WHERE id = $2)
				OR (created_at = (SELECT created_at FROM bikes WHERE id = $2) AND id < $2)
			)
			ORDER BY created_at DESC, id DESC
			LIMIT $3
		`
		args = []any{userID, cursor, limit + 1}
	} else {
		query = `
			SELECT 
				id, user_id, registration_number,
				COALESCE(label, ''), COALESCE(make, ''), COALESCE(model, ''), COALESCE(year, 2024), COALESCE(color, ''), imei,
				COALESCE(status, 'STOPPED'), COALESCE(ignition, false), COALESCE(speed_kmh, 0.0), COALESCE(heading_deg, 0.0),
				COALESCE(latitude, 0.0), COALESCE(longitude, 0.0),
				COALESCE(current_address, ''), COALESCE(sub_county, ''), COALESCE(county, ''),
				COALESCE(battery_percent, 100), COALESCE(signal_quality_csq, 20), last_telemetry_at,
				COALESCE(geofence_id, ''), COALESCE(driver_contact_id, ''),
				COALESCE(is_immobilized, false), created_at, updated_at
			FROM bikes
			WHERE user_id = $1
			ORDER BY created_at DESC, id DESC
			LIMIT $2
		`
		args = []any{userID, limit + 1}
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, domain.BikeListMeta{}, err
	}
	defer rows.Close()

	var bikes []domain.Bike
	for rows.Next() {
		var b domain.Bike
		err := rows.Scan(
			&b.ID, &b.UserID, &b.RegistrationNumber,
			&b.Label, &b.Make, &b.Model, &b.Year, &b.Color, &b.IMEI,
			&b.Status, &b.Ignition, &b.SpeedKMH, &b.HeadingDeg,
			&b.Latitude, &b.Longitude,
			&b.CurrentAddress, &b.SubCounty, &b.County,
			&b.BatteryPercent, &b.SignalQualityCSQ, &b.LastTelemetryAt,
			&b.GeofenceID, &b.DriverContactID,
			&b.IsImmobilized, &b.CreatedAt, &b.UpdatedAt,
		)
		if err != nil {
			return nil, domain.BikeListMeta{}, err
		}
		bikes = append(bikes, b)
	}
	if err := rows.Err(); err != nil {
		return nil, domain.BikeListMeta{}, err
	}

	hasMore := len(bikes) > limit
	nextCursor := ""
	if hasMore {
		nextCursor = bikes[limit-1].ID
		bikes = bikes[:limit]
	}

	meta := domain.BikeListMeta{
		Total:      total,
		Limit:      limit,
		Cursor:     cursor,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}

	return bikes, meta, nil
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
			updated_at = $13
		WHERE user_id = $14 AND id = $15
	`
	tag, err := tx.Exec(ctx, query,
		b.RegistrationNumber, b.Label, b.Make, b.Model, b.Year, b.Color,
		b.Status, b.CurrentAddress, b.SubCounty, b.County,
		geofenceVal, driverVal,
		b.UpdatedAt,
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
	var row pgx.Row
	if excludeBikeID != "" {
		query := `SELECT 1 FROM bikes WHERE imei = $1 AND id != $2 LIMIT 1`
		row = r.pool.QueryRow(ctx, query, imei, excludeBikeID)
	} else {
		query := `SELECT 1 FROM bikes WHERE imei = $1 LIMIT 1`
		row = r.pool.QueryRow(ctx, query, imei)
	}
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
	var row pgx.Row
	if excludeBikeID != "" {
		query := `SELECT 1 FROM bikes WHERE user_id = $1 AND LOWER(registration_number) = LOWER($2) AND id != $3 LIMIT 1`
		row = r.pool.QueryRow(ctx, query, userID, regNumber, excludeBikeID)
	} else {
		query := `SELECT 1 FROM bikes WHERE user_id = $1 AND LOWER(registration_number) = LOWER($2) LIMIT 1`
		row = r.pool.QueryRow(ctx, query, userID, regNumber)
	}
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

func (r *BikeRepository) GetBreadcrumbs(
	ctx context.Context,
	userID string,
	bikeID string,
	startDate string,
	endDate string,
	cursor int64,
	limit int,
) ([]domain.BreadcrumbDTO, domain.PaginationMeta, error) {
	if limit <= 0 || limit > 1000 {
		limit = 25
	}

	var total int
	countQuery := `
		SELECT COUNT(*)
		FROM bike_breadcrumbs bb
		JOIN bikes b ON b.id = bb.bike_id
		WHERE b.user_id = $1 AND bb.bike_id = $2
	`
	var countArgs []any
	if startDate != "" && endDate != "" {
		countQuery += ` AND bb.recorded_at >= $3 AND bb.recorded_at <= $4`
		countArgs = []any{userID, bikeID, startDate, endDate}
	} else if startDate != "" {
		countQuery += ` AND bb.recorded_at >= $3`
		countArgs = []any{userID, bikeID, startDate}
	} else {
		countArgs = []any{userID, bikeID}
	}
	_ = r.pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total)

	whereClause := "b.user_id = $1 AND bb.bike_id = $2"
	args := []any{userID, bikeID}
	argIdx := 3

	if startDate != "" && endDate != "" {
		whereClause += fmt.Sprintf(" AND bb.recorded_at >= $%d AND bb.recorded_at <= $%d", argIdx, argIdx+1)
		args = append(args, startDate, endDate)
		argIdx += 2
	} else if startDate != "" {
		whereClause += fmt.Sprintf(" AND bb.recorded_at >= $%d", argIdx)
		args = append(args, startDate)
		argIdx++
	}

	if cursor > 0 {
		whereClause += fmt.Sprintf(" AND bb.id > $%d", argIdx)
		args = append(args, cursor)
		argIdx++
	}

	query := fmt.Sprintf(`
		SELECT bb.id, bb.bike_id, bb.latitude, bb.longitude, bb.speed_kmh, bb.heading_deg, bb.ignition, bb.battery_percent, bb.address, bb.geofence_breach, bb.recorded_at
		FROM bike_breadcrumbs bb
		JOIN bikes b ON b.id = bb.bike_id
		WHERE %s
		ORDER BY bb.id ASC
		LIMIT $%d
	`, whereClause, argIdx)
	args = append(args, limit+1)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, domain.PaginationMeta{}, err
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
			return nil, domain.PaginationMeta{}, err
		}
		points = append(points, p)
	}
	if err := rows.Err(); err != nil {
		return nil, domain.PaginationMeta{}, err
	}

	hasMore := len(points) > limit
	var nextCursor int64
	if hasMore {
		nextCursor = points[limit-1].ID
		points = points[:limit]
	}

	cursorStr := ""
	if cursor > 0 {
		cursorStr = strconv.FormatInt(cursor, 10)
	}
	nextCursorStr := ""
	if nextCursor > 0 {
		nextCursorStr = strconv.FormatInt(nextCursor, 10)
	}

	meta := domain.PaginationMeta{
		Total:      total,
		Limit:      limit,
		Cursor:     cursorStr,
		NextCursor: nextCursorStr,
		HasMore:    hasMore,
	}

	return points, meta, nil
}

func (r *BikeRepository) GetEvents(
	ctx context.Context,
	userID string,
	bikeID string,
	startDate string,
	endDate string,
	cursor int64,
	limit int,
) ([]domain.BikeEventDTO, domain.PaginationMeta, error) {
	if limit <= 0 || limit > 1000 {
		limit = 25
	}

	var total int
	countQuery := `
		SELECT COUNT(*)
		FROM bike_events be
		JOIN bikes b ON b.id = be.bike_id
		WHERE b.user_id = $1 AND be.bike_id = $2
	`
	var countArgs []any
	if startDate != "" && endDate != "" {
		countQuery += ` AND be.recorded_at >= $3 AND be.recorded_at <= $4`
		countArgs = []any{userID, bikeID, startDate, endDate}
	} else if startDate != "" {
		countQuery += ` AND be.recorded_at >= $3`
		countArgs = []any{userID, bikeID, startDate}
	} else {
		countArgs = []any{userID, bikeID}
	}
	_ = r.pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total)

	whereClause := "b.user_id = $1 AND be.bike_id = $2"
	args := []any{userID, bikeID}
	argIdx := 3

	if startDate != "" && endDate != "" {
		whereClause += fmt.Sprintf(" AND be.recorded_at >= $%d AND be.recorded_at <= $%d", argIdx, argIdx+1)
		args = append(args, startDate, endDate)
		argIdx += 2
	} else if startDate != "" {
		whereClause += fmt.Sprintf(" AND be.recorded_at >= $%d", argIdx)
		args = append(args, startDate)
		argIdx++
	}

	if cursor > 0 {
		whereClause += fmt.Sprintf(" AND be.id > $%d", argIdx)
		args = append(args, cursor)
		argIdx++
	}

	query := fmt.Sprintf(`
		SELECT be.id, be.bike_id, be.event_type, be.latitude, be.longitude, be.recorded_at
		FROM bike_events be
		JOIN bikes b ON b.id = be.bike_id
		WHERE %s
		ORDER BY be.id ASC
		LIMIT $%d
	`, whereClause, argIdx)
	args = append(args, limit+1)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, domain.PaginationMeta{}, err
	}
	defer rows.Close()

	var events []domain.BikeEventDTO
	for rows.Next() {
		var e domain.BikeEventDTO
		err := rows.Scan(
			&e.ID, &e.BikeID, &e.EventType, &e.Latitude, &e.Longitude, &e.RecordedAt,
		)
		if err != nil {
			return nil, domain.PaginationMeta{}, err
		}
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		return nil, domain.PaginationMeta{}, err
	}

	hasMore := len(events) > limit
	var nextCursor int64
	if hasMore {
		nextCursor = events[limit-1].ID
		events = events[:limit]
	}

	cursorStr := ""
	if cursor > 0 {
		cursorStr = strconv.FormatInt(cursor, 10)
	}
	nextCursorStr := ""
	if nextCursor > 0 {
		nextCursorStr = strconv.FormatInt(nextCursor, 10)
	}

	meta := domain.PaginationMeta{
		Total:      total,
		Limit:      limit,
		Cursor:     cursorStr,
		NextCursor: nextCursorStr,
		HasMore:    hasMore,
	}

	return events, meta, nil
}
