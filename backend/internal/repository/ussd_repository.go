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

// GetBikesByPhone retrieves all motorcycles linked to the caller's phone number directly via users.
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
			COALESCE(u.name, 'Owner'),
			COALESCE(u.name, 'Rider')
		FROM bikes b
		JOIN users u ON b.user_id = u.id
		WHERE 
			-- Match User Phone directly
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

// CreatePendingCommand queues an immobilize or reactivate command for a bike from USSD.
// The bike's state changes only when the device confirms the command.
func (r *USSDRepository) CreatePendingCommand(ctx context.Context, bikeID string, immobilize bool, ts int64) (imei string, userID string, err error) {
	return createPendingCommand(ctx, r.pool, "", bikeID, immobilize, ts)
}

// RecordEmergencySOS creates a high-priority security SOS alert record in the database.
func (r *USSDRepository) RecordEmergencySOS(ctx context.Context, bikeID, userID string, lat, lng float64, details string) error {
	query := `
		INSERT INTO bike_events (bike_id, event_type, latitude, longitude, recorded_at)
		VALUES ($1, 'SOS_DURESS', $2, $3, NOW())
	`
	_, err := r.pool.Exec(ctx, query, bikeID, lat, lng)
	return err
}

type RegisterBikeParams struct {
	Phone              string
	RegistrationNumber string
	IMEI               string
	Make               string
	Model              string
}

// RegisterBike creates a new motorcycle record linked to the caller's phone number.
func (r *USSDRepository) RegisterBike(ctx context.Context, params RegisterBikeParams) (*domain.RiderBikeInfo, error) {
	normPhone, last9 := NormalizePhoneNumber(params.Phone)
	regNumber := strings.ToUpper(strings.TrimSpace(params.RegistrationNumber))
	imei := strings.TrimSpace(params.IMEI)

	if regNumber == "" {
		return nil, errors.New("registration number is required")
	}
	if imei == "" {
		return nil, errors.New("tracker IMEI is required")
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("transaction begin error: %w", err)
	}
	defer tx.Rollback(ctx)

	// Check if registration number already registered
	var existingReg string
	err = tx.QueryRow(ctx, "SELECT registration_number FROM bikes WHERE LOWER(registration_number) = LOWER($1)", regNumber).Scan(&existingReg)
	if err == nil {
		return nil, fmt.Errorf("motorcycle %s is already registered in the system", regNumber)
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("check registration error: %w", err)
	}

	// Check if IMEI already registered
	var existingIMEI string
	err = tx.QueryRow(ctx, "SELECT imei FROM bikes WHERE imei = $1", imei).Scan(&existingIMEI)
	if err == nil {
		return nil, fmt.Errorf("tracker IMEI %s is already registered to another motorcycle", imei)
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("check imei error: %w", err)
	}

	// Find or create user
	var userID string
	var userName string
	userQuery := `
		SELECT id::text, COALESCE(name, 'Rider')
		FROM users
		WHERE phone_number = $1 OR phone_number = $2 OR phone_number LIKE '%' || $3
		LIMIT 1
	`
	err = tx.QueryRow(ctx, userQuery, params.Phone, normPhone, last9).Scan(&userID, &userName)
	if errors.Is(err, pgx.ErrNoRows) {
		riderName := "Rider " + regNumber
		userName = riderName
		digits := ""
		for _, ch := range normPhone {
			if ch >= '0' && ch <= '9' {
				digits += string(ch)
			}
		}
		email := digits + "@mguard.ke"
		insertUserQuery := `
			INSERT INTO users (id, email, password_hash, name, phone_number, created_at, updated_at)
			VALUES (gen_random_uuid(), $1, '$2a$10$temporaryHashForUSSDAccount', $2, $3, NOW(), NOW())
			RETURNING id::text
		`
		if err := tx.QueryRow(ctx, insertUserQuery, email, riderName, normPhone).Scan(&userID); err != nil {
			return nil, fmt.Errorf("create rider user error: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("lookup rider user error: %w", err)
	}

	// Find or create driver_contact
	var driverContactID string
	dcQuery := `
		SELECT id::text
		FROM driver_contacts
		WHERE user_id = $1::uuid AND (phone = $2 OR phone = $3 OR phone LIKE '%' || $4)
		LIMIT 1
	`
	err = tx.QueryRow(ctx, dcQuery, userID, params.Phone, normPhone, last9).Scan(&driverContactID)
	if errors.Is(err, pgx.ErrNoRows) {
		insertDCQuery := `
			INSERT INTO driver_contacts (id, user_id, name, phone, created_at, updated_at)
			VALUES (gen_random_uuid(), $1::uuid, $2, $3, NOW(), NOW())
			RETURNING id::text
		`
		if err := tx.QueryRow(ctx, insertDCQuery, userID, userName, normPhone).Scan(&driverContactID); err != nil {
			return nil, fmt.Errorf("create driver contact error: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("lookup driver contact error: %w", err)
	}

	// Ensure tracker exists in trackers table
	_, _ = tx.Exec(ctx, `
		INSERT INTO trackers (id, imei, model, status, created_at, updated_at)
		VALUES (gen_random_uuid(), $1, 'MGuard GPS', 'ACTIVE', NOW(), NOW())
		ON CONFLICT (imei) DO NOTHING
	`, imei)

	// Determine make and model
	bikeMake := params.Make
	if bikeMake == "" {
		bikeMake = "Bajaj"
	}
	bikeModel := params.Model
	if bikeModel == "" {
		bikeModel = "Boxer 150"
	}
	label := bikeMake + " " + bikeModel

	var bikeID string
	insertBikeQuery := `
		INSERT INTO bikes (
			id, user_id, registration_number, label, make, model, year,
			imei, status, ignition, speed_kmh, latitude, longitude,
			current_address, driver_contact_id, is_immobilized, battery_percent,
			created_at, updated_at
		)
		VALUES (
			gen_random_uuid(), $1::uuid, $2, $3, $4, $5, 2024,
			$6, 'PARKED', false, 0.0, -1.2921, 36.8219,
			'Registered via MGuard USSD', $7::uuid, false, 100,
			NOW(), NOW()
		)
		RETURNING id::text
	`
	err = tx.QueryRow(ctx, insertBikeQuery, userID, regNumber, label, bikeMake, bikeModel, imei, driverContactID).Scan(&bikeID)
	if err != nil {
		return nil, fmt.Errorf("insert bike error: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit error: %w", err)
	}

	return &domain.RiderBikeInfo{
		ID:                 bikeID,
		UserID:             userID,
		RegistrationNumber: regNumber,
		Label:              label,
		Make:               bikeMake,
		Model:              bikeModel,
		Status:             "PARKED",
		Ignition:           false,
		SpeedKMH:           0,
		Latitude:           -1.2921,
		Longitude:          36.8219,
		CurrentAddress:     "Registered via MGuard USSD",
		BatteryPercent:     100,
		IsImmobilized:      false,
		IMEI:               imei,
	}, nil
}
