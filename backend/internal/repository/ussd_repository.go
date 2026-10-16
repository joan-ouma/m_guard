package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"motoguard-backend/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type USSDRepository struct {
	pool *pgxpool.Pool
}

func NewUSSDRepository(pool *pgxpool.Pool) *USSDRepository {
	return &USSDRepository{pool: pool}
}

// NormalizePhoneNumber normalizes standard Kenyan/international phone numbers into E.164 and suffix patterns.
func NormalizePhoneNumber(phone string) (normalized string, last9 string) {
	digits := ""
	for _, ch := range phone {
		if ch >= '0' && ch <= '9' {
			digits += string(ch)
		}
	}

	if len(digits) >= 9 {
		last9 = digits[len(digits)-9:]
	} else {
		last9 = digits
	}

	if strings.HasPrefix(digits, "254") && len(digits) == 12 {
		normalized = "+" + digits
	} else if strings.HasPrefix(digits, "0") && len(digits) == 10 {
		normalized = "+254" + digits[1:]
	} else if len(digits) == 9 {
		normalized = "+254" + digits
	} else if strings.HasPrefix(phone, "+") {
		normalized = "+" + digits
	} else {
		normalized = "+" + digits
	}

	return normalized, last9
}

// GetBikesByPhone retrieves all motorcycles linked to the caller's phone number
// either as an assigned rider (driver_contacts) or as the fleet owner (users).
func (r *USSDRepository) GetBikesByPhone(ctx context.Context, phone string) ([]domain.RiderBikeInfo, error) {
	normalized, last9 := NormalizePhoneNumber(phone)

	query := `
		SELECT DISTINCT
			b.id::text,
			b.user_id::text,
			b.registration_number,
			COALESCE(b.label, ''),
			COALESCE(b.make, ''),
			COALESCE(b.model, ''),
			COALESCE(b.status, 'STOPPED'),
			COALESCE(b.ignition, false),
			COALESCE(b.speed_kmh, 0.0),
			COALESCE(b.latitude, 0.0),
			COALESCE(b.longitude, 0.0),
			COALESCE(b.current_address, 'Location pending update'),
			COALESCE(b.sub_county, ''),
			COALESCE(b.county, ''),
			COALESCE(b.battery_percent, 100),
			COALESCE(b.signal_quality_csq, 20),
			COALESCE(b.is_immobilized, false),
			b.imei,
			b.last_telemetry_at,
			COALESCE(u.phone_number, ''),
			COALESCE(u.name, 'Fleet Owner'),
			COALESCE(dc.name, 'Rider')
		FROM bikes b
		JOIN users u ON b.user_id = u.id
		LEFT JOIN driver_contacts dc ON b.driver_contact_id = dc.id
		WHERE 
			-- Match Driver Contact Phone
			(dc.phone = $1 OR dc.phone = $2 OR dc.phone LIKE '%' || $3)
			OR
			-- Match Fleet Owner Phone
			(u.phone_number = $1 OR u.phone_number = $2 OR u.phone_number LIKE '%' || $3)
		ORDER BY b.registration_number ASC
	`

	rows, err := r.pool.Query(ctx, query, phone, normalized, last9)
	if err != nil {
		return nil, fmt.Errorf("failed to query bikes by phone: %w", err)
	}
	defer rows.Close()

	var bikes []domain.RiderBikeInfo
	for rows.Next() {
		var bike domain.RiderBikeInfo
		err := rows.Scan(
			&bike.ID,
			&bike.UserID,
			&bike.RegistrationNumber,
			&bike.Label,
			&bike.Make,
			&bike.Model,
			&bike.Status,
			&bike.Ignition,
			&bike.SpeedKMH,
			&bike.Latitude,
			&bike.Longitude,
			&bike.CurrentAddress,
			&bike.SubCounty,
			&bike.County,
			&bike.BatteryPercent,
			&bike.SignalQualityCSQ,
			&bike.IsImmobilized,
			&bike.IMEI,
			&bike.LastTelemetryAt,
			&bike.OwnerPhone,
			&bike.OwnerName,
			&bike.RiderName,
		)
		if err != nil {
			return nil, fmt.Errorf("scan bike row error: %w", err)
		}
		bikes = append(bikes, bike)
	}

	return bikes, nil
}

// GetBikeByRegOrID finds a single motorcycle by registration number or ID for the caller's phone.
func (r *USSDRepository) GetBikeByRegOrID(ctx context.Context, phone string, identifier string) (*domain.RiderBikeInfo, error) {
	bikes, err := r.GetBikesByPhone(ctx, phone)
	if err != nil {
		return nil, err
	}

	cleanIdent := strings.ToUpper(strings.ReplaceAll(identifier, " ", ""))
	for _, b := range bikes {
		cleanReg := strings.ToUpper(strings.ReplaceAll(b.RegistrationNumber, " ", ""))
		if cleanReg == cleanIdent || b.ID == identifier {
			return &b, nil
		}
	}

	return nil, ErrNotFound
}

// GetRiderPIN fetches stored security PIN record for a phone number.
func (r *USSDRepository) GetRiderPIN(ctx context.Context, phone string) (*domain.RiderPIN, error) {
	normalized, _ := NormalizePhoneNumber(phone)

	query := `
		SELECT phone_number, pin_hash, failed_attempts, locked_until, created_at, updated_at
		FROM ussd_pins
		WHERE phone_number = $1 OR phone_number = $2
		LIMIT 1
	`

	var p domain.RiderPIN
	err := r.pool.QueryRow(ctx, query, phone, normalized).Scan(
		&p.PhoneNumber,
		&p.PINHash,
		&p.FailedAttempts,
		&p.LockedUntil,
		&p.CreatedAt,
		&p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get rider pin error: %w", err)
	}

	return &p, nil
}

// SaveRiderPIN creates or updates the security PIN for a phone number.
func (r *USSDRepository) SaveRiderPIN(ctx context.Context, phone, pinHash string) error {
	normalized, _ := NormalizePhoneNumber(phone)

	query := `
		INSERT INTO ussd_pins (phone_number, pin_hash, failed_attempts, locked_until, created_at, updated_at)
		VALUES ($1, $2, 0, NULL, NOW(), NOW())
		ON CONFLICT (phone_number) DO UPDATE
		SET pin_hash = EXCLUDED.pin_hash,
		    failed_attempts = 0,
		    locked_until = NULL,
		    updated_at = NOW()
	`

	_, err := r.pool.Exec(ctx, query, normalized, pinHash)
	if err != nil {
		return fmt.Errorf("save rider pin error: %w", err)
	}

	return nil
}

// RecordFailedPINAttempt updates failed attempt counter and locks out if threshold exceeded.
func (r *USSDRepository) RecordFailedPINAttempt(ctx context.Context, phone string, attempts int, lockedUntil *time.Time) error {
	normalized, _ := NormalizePhoneNumber(phone)

	query := `
		UPDATE ussd_pins
		SET failed_attempts = $1,
		    locked_until = $2,
		    updated_at = NOW()
		WHERE phone_number = $3
	`

	_, err := r.pool.Exec(ctx, query, attempts, lockedUntil, normalized)
	return err
}

// ResetFailedPINAttempts clears failed attempts upon successful PIN entry.
func (r *USSDRepository) ResetFailedPINAttempts(ctx context.Context, phone string) error {
	normalized, _ := NormalizePhoneNumber(phone)

	query := `
		UPDATE ussd_pins
		SET failed_attempts = 0,
		    locked_until = NULL,
		    updated_at = NOW()
		WHERE phone_number = $1
	`

	_, err := r.pool.Exec(ctx, query, normalized)
	return err
}

// SetBikeImmobilizedByRider immobilizes or mobilizes motorcycle directly via USSD/SMS.
func (r *USSDRepository) SetBikeImmobilizedByRider(ctx context.Context, bikeID string, immobilize bool, triggeredBy string) (imei string, userID string, err error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return "", "", err
	}
	defer tx.Rollback(ctx)

	var currentImmobilized bool
	var lat, lng float64

	checkQuery := `
		SELECT is_immobilized, imei, user_id::text, latitude, longitude
		FROM bikes
		WHERE id = $1
		FOR UPDATE
	`
	err = tx.QueryRow(ctx, checkQuery, bikeID).Scan(&currentImmobilized, &imei, &userID, &lat, &lng)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", ErrNotFound
		}
		return "", "", err
	}

	updateQuery := `
		UPDATE bikes
		SET is_immobilized = $1, updated_at = NOW()
		WHERE id = $2
	`
	if _, err := tx.Exec(ctx, updateQuery, immobilize, bikeID); err != nil {
		return "", "", err
	}

	eventType := "IMMOBILIZED"
	if !immobilize {
		eventType = "REACTIVATED"
	}

	description := fmt.Sprintf("Engine %s via %s", eventType, triggeredBy)
	eventQuery := `
		INSERT INTO bike_events (id, bike_id, event_type, latitude, longitude, recorded_at, description)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, NOW(), $5)
	`
	if _, err := tx.Exec(ctx, eventQuery, bikeID, eventType, lat, lng, description); err != nil {
		return "", "", err
	}

	if err := tx.Commit(ctx); err != nil {
		return "", "", err
	}

	return imei, userID, nil
}

// RecordEmergencySOS creates a high-priority security SOS alert record in the database.
func (r *USSDRepository) RecordEmergencySOS(ctx context.Context, bikeID, userID string, lat, lng float64, details string) error {
	query := `
		INSERT INTO bike_events (id, bike_id, event_type, latitude, longitude, recorded_at, description)
		VALUES (gen_random_uuid(), $1, 'SOS_DURESS', $2, $3, NOW(), $4)
	`
	_, err := r.pool.Exec(ctx, query, bikeID, lat, lng, details)
	return err
}
