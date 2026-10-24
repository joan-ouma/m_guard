package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrCommandPending = errors.New("a command is already awaiting confirmation")
	ErrAlreadyInState = errors.New("bike is already in the requested state")
)

// PendingCommand is an immobilizer command the device has not confirmed yet.
type PendingCommand struct {
	ID       string
	BikeID   string
	IMEI     string
	Action   string
	TS       int64
	Attempts int
}

// CreatePendingCommand queues an immobilize or reactivate command without changing the bike's state.
// The state changes only when the device confirms the command. userID may be empty for USSD riders,
// who are identified by bike ID alone. It returns the bike's IMEI and owner.
func (r *BikeRepository) CreatePendingCommand(ctx context.Context, userID, bikeID string, immobilize bool, ts int64) (string, string, error) {
	return createPendingCommand(ctx, r.pool, userID, bikeID, immobilize, ts)
}

func createPendingCommand(ctx context.Context, pool *pgxpool.Pool, userID, bikeID string, immobilize bool, ts int64) (string, string, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return "", "", err
	}
	defer tx.Rollback(ctx)

	// Locking the bike row serializes concurrent requests for the same bike.
	var currentImmobilized bool
	var imei, ownerID string
	lockQuery := `
		SELECT COALESCE(is_immobilized, false), imei, user_id::text
		FROM bikes
		WHERE id = $1 AND ($2 = '' OR user_id::text = $2)
		FOR UPDATE
	`
	err = tx.QueryRow(ctx, lockQuery, bikeID, userID).Scan(&currentImmobilized, &imei, &ownerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", ErrNotFound
		}
		return "", "", err
	}

	if currentImmobilized == immobilize {
		return "", "", ErrAlreadyInState
	}

	action := "IMMOBILIZE"
	if !immobilize {
		action = "REACTIVATE"
	}

	insertQuery := `
		INSERT INTO bike_commands (bike_id, imei, action, ts)
		VALUES ($1, $2, $3, $4)
	`
	if _, err := tx.Exec(ctx, insertQuery, bikeID, imei, action, ts); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return "", "", ErrCommandPending
		}
		return "", "", err
	}

	if err := tx.Commit(ctx); err != nil {
		return "", "", err
	}

	return imei, ownerID, nil
}

// ConfirmCommand applies the device's confirmation to the pending command with the same timestamp.
// It returns the bike ID and true only when the confirmation matched a pending command.
func (r *BikeRepository) ConfirmCommand(ctx context.Context, imei string, ts int64, immobilized bool) (string, bool, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return "", false, err
	}
	defer tx.Rollback(ctx)

	var commandID, bikeID, action string
	findQuery := `
		SELECT id::text, bike_id::text, action
		FROM bike_commands
		WHERE imei = $1 AND ts = $2 AND status = 'PENDING'
		FOR UPDATE
	`
	err = tx.QueryRow(ctx, findQuery, imei, ts).Scan(&commandID, &bikeID, &action)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", false, nil
		}
		return "", false, err
	}

	// A confirmation that reports the wrong state is ignored, so the command keeps retrying.
	if (action == "IMMOBILIZE") != immobilized {
		return "", false, nil
	}

	if _, err := tx.Exec(ctx, `UPDATE bikes SET is_immobilized = $1, updated_at = NOW() WHERE id = $2`, immobilized, bikeID); err != nil {
		return "", false, err
	}

	eventType := "IMMOBILIZED"
	if !immobilized {
		eventType = "REACTIVATED"
	}
	eventQuery := `
		INSERT INTO bike_events (bike_id, event_type, latitude, longitude, recorded_at)
		SELECT id, $2, COALESCE(latitude, 0), COALESCE(longitude, 0), NOW()
		FROM bikes
		WHERE id = $1
	`
	if _, err := tx.Exec(ctx, eventQuery, bikeID, eventType); err != nil {
		return "", false, err
	}

	if _, err := tx.Exec(ctx, `UPDATE bike_commands SET status = 'CONFIRMED', confirmed_at = NOW() WHERE id = $1`, commandID); err != nil {
		return "", false, err
	}

	if err := tx.Commit(ctx); err != nil {
		return "", false, err
	}

	return bikeID, true, nil
}

// ListStalePendingCommands returns pending commands not sent again within the given age.
func (r *BikeRepository) ListStalePendingCommands(ctx context.Context, olderThan time.Duration) ([]PendingCommand, error) {
	query := `
		SELECT id::text, bike_id::text, imei, action, ts, attempts
		FROM bike_commands
		WHERE status = 'PENDING' AND last_sent_at < NOW() - make_interval(secs => $1)
		ORDER BY last_sent_at
	`
	rows, err := r.pool.Query(ctx, query, olderThan.Seconds())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var commands []PendingCommand
	for rows.Next() {
		var c PendingCommand
		if err := rows.Scan(&c.ID, &c.BikeID, &c.IMEI, &c.Action, &c.TS, &c.Attempts); err != nil {
			return nil, err
		}
		commands = append(commands, c)
	}
	return commands, rows.Err()
}

// MarkCommandResent records another send of a pending command.
func (r *BikeRepository) MarkCommandResent(ctx context.Context, commandID string) error {
	_, err := r.pool.Exec(ctx, `UPDATE bike_commands SET attempts = attempts + 1, last_sent_at = NOW() WHERE id = $1`, commandID)
	return err
}

// MarkCommandFailed closes a command the device never confirmed. The bike keeps its previous state.
func (r *BikeRepository) MarkCommandFailed(ctx context.Context, commandID string) error {
	_, err := r.pool.Exec(ctx, `UPDATE bike_commands SET status = 'FAILED' WHERE id = $1`, commandID)
	return err
}
