package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"motoguard-backend/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("record not found")

type AuthRepository struct {
	pool *pgxpool.Pool
}

func NewAuthRepository(pool *pgxpool.Pool) *AuthRepository {
	return &AuthRepository{pool: pool}
}

func (r *AuthRepository) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, email, password_hash, COALESCE(name, ''), COALESCE(national_id, ''), COALESCE(phone_number, ''), COALESCE(created_at, NOW()), COALESCE(updated_at, NOW()), deletion_scheduled_at
		FROM users
		WHERE LOWER(email) = LOWER($1)
		LIMIT 1
	`
	row := r.pool.QueryRow(ctx, query, email)

	var u domain.User
	var nationalID, phone sql.NullString
	var scheduled sql.NullTime
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &nationalID, &phone, &u.CreatedAt, &u.UpdatedAt, &scheduled)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if nationalID.Valid {
		u.NationalID = nationalID.String
	}
	if phone.Valid {
		u.PhoneNumber = phone.String
	}
	if scheduled.Valid {
		t := scheduled.Time
		u.DeletionScheduledAt = &t
	}
	return &u, nil
}

func (r *AuthRepository) CreateUser(ctx context.Context, u *domain.User) error {
	query := `
		INSERT INTO users (id, email, password_hash, name, national_id, phone_number, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.pool.Exec(ctx, query,
		u.ID,
		u.Email,
		u.PasswordHash,
		u.Name,
		u.NationalID,
		u.PhoneNumber,
		u.CreatedAt,
		u.UpdatedAt,
	)
	return err
}

func (r *AuthRepository) GetUserByID(ctx context.Context, userID string) (*domain.User, error) {
	query := `
		SELECT id, email, password_hash, COALESCE(name, ''), COALESCE(national_id, ''), COALESCE(phone_number, ''), COALESCE(created_at, NOW()), COALESCE(updated_at, NOW()), deletion_scheduled_at
		FROM users
		WHERE id = $1
		LIMIT 1
	`
	row := r.pool.QueryRow(ctx, query, userID)

	var u domain.User
	var nationalID, phone sql.NullString
	var scheduled sql.NullTime
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &nationalID, &phone, &u.CreatedAt, &u.UpdatedAt, &scheduled)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if nationalID.Valid {
		u.NationalID = nationalID.String
	}
	if phone.Valid {
		u.PhoneNumber = phone.String
	}
	if scheduled.Valid {
		t := scheduled.Time
		u.DeletionScheduledAt = &t
	}
	return &u, nil
}

func (r *AuthRepository) GetSessionWithUser(ctx context.Context, tokenHash string) (*domain.Session, *domain.User, error) {
	query := `
		SELECT 
			s.id, s.user_id, s.refresh_token_hash, s.expires_at, COALESCE(s.revoked_at, '0001-01-01 00:00:00+00'), s.user_agent, s.ip_address, s.created_at,
			u.id, u.email, u.password_hash, u.name, u.national_id, u.phone_number, u.created_at, u.updated_at
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.refresh_token_hash = $1
		LIMIT 1
	`
	row := r.pool.QueryRow(ctx, query, tokenHash)

	var s domain.Session
	var revokedAt time.Time
	var u domain.User
	var nationalID, phone sql.NullString

	err := row.Scan(
		&s.ID, &s.UserID, &s.RefreshTokenHash, &s.ExpiresAt, &revokedAt, &s.UserAgent, &s.IPAddress, &s.CreatedAt,
		&u.ID, &u.Email, &u.PasswordHash, &u.Name, &nationalID, &phone, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, ErrNotFound
		}
		return nil, nil, err
	}

	if !revokedAt.IsZero() && revokedAt.Year() > 1 {
		s.RevokedAt = revokedAt
	}
	if nationalID.Valid {
		u.NationalID = nationalID.String
	}
	if phone.Valid {
		u.PhoneNumber = phone.String
	}

	return &s, &u, nil
}

func (r *AuthRepository) CreateSession(ctx context.Context, tx pgx.Tx, session *domain.Session) error {
	query := `
		INSERT INTO sessions (id, user_id, refresh_token_hash, expires_at, user_agent, ip_address, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := tx.Exec(ctx, query,
		session.ID,
		session.UserID,
		session.RefreshTokenHash,
		session.ExpiresAt,
		session.UserAgent,
		session.IPAddress,
		session.CreatedAt,
	)
	return err
}

func (r *AuthRepository) RevokeSession(ctx context.Context, tx pgx.Tx, sessionID string) error {
	query := `UPDATE sessions SET revoked_at = NOW() WHERE id = $1 AND revoked_at IS NULL`
	_, err := tx.Exec(ctx, query, sessionID)
	return err
}

func (r *AuthRepository) CreatePasswordResetToken(ctx context.Context, tx pgx.Tx, prt *domain.PasswordResetToken) error {
	query := `
		INSERT INTO password_reset_tokens (id, user_id, token_hash, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := tx.Exec(ctx, query,
		prt.ID,
		prt.UserID,
		prt.TokenHash,
		prt.ExpiresAt,
		prt.CreatedAt,
	)
	return err
}

func (r *AuthRepository) GetPasswordResetToken(ctx context.Context, tokenHash string) (*domain.PasswordResetToken, error) {
	query := `
		SELECT id, user_id, token_hash, expires_at, COALESCE(used_at, '0001-01-01 00:00:00+00'), created_at
		FROM password_reset_tokens
		WHERE token_hash = $1
	`
	row := r.pool.QueryRow(ctx, query, tokenHash)

	var prt domain.PasswordResetToken
	var usedAt time.Time
	err := row.Scan(
		&prt.ID,
		&prt.UserID,
		&prt.TokenHash,
		&prt.ExpiresAt,
		&usedAt,
		&prt.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if !usedAt.IsZero() && usedAt.Year() > 1 {
		prt.UsedAt = usedAt
	}

	return &prt, nil
}

func (r *AuthRepository) MarkPasswordResetUsed(ctx context.Context, tx pgx.Tx, tokenID string) error {
	query := `UPDATE password_reset_tokens SET used_at = NOW() WHERE id = $1`
	_, err := tx.Exec(ctx, query, tokenID)
	return err
}

func (r *AuthRepository) UpdateUserPassword(ctx context.Context, tx pgx.Tx, userID string, newPasswordHash string) error {
	query := `UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2`
	_, err := tx.Exec(ctx, query, newPasswordHash, userID)
	return err
}

func (r *AuthRepository) RevokeAllUserSessions(ctx context.Context, tx pgx.Tx, userID string) error {
	query := `UPDATE sessions SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`
	_, err := tx.Exec(ctx, query, userID)
	return err
}

// AccountDeletionGrace is how long a deletion waits before it is carried out.
const AccountDeletionGrace = 30 * 24 * time.Hour

// DeleteAccount removes a user and everything they own, inside the caller's transaction.
// Tables with no foreign key to users are cleared explicitly; the rest cascade from users.
func (r *AuthRepository) DeleteAccount(ctx context.Context, tx pgx.Tx, userID string) error {
	statements := []string{
		`DELETE FROM chat_messages WHERE user_id = $1`,
		`DELETE FROM theft_incidents WHERE user_id = $1`,
		`DELETE FROM user_contacts WHERE user_id = $1`,
		`DELETE FROM driver_contacts WHERE user_id = $1`,
		`DELETE FROM users WHERE id = $1`,
	}
	for _, stmt := range statements {
		if _, err := tx.Exec(ctx, stmt, userID); err != nil {
			return err
		}
	}
	return nil
}

// ScheduleAccountDeletion marks the account for deletion and returns when it will be carried out.
func (r *AuthRepository) ScheduleAccountDeletion(ctx context.Context, userID string) (time.Time, error) {
	var scheduledFor time.Time
	err := r.pool.QueryRow(ctx,
		`UPDATE users SET deletion_scheduled_at = COALESCE(deletion_scheduled_at, NOW()) WHERE id = $1 RETURNING deletion_scheduled_at`,
		userID).Scan(&scheduledFor)
	if err != nil {
		return time.Time{}, err
	}
	return scheduledFor.Add(AccountDeletionGrace), nil
}

// CancelAccountDeletion clears a pending deletion, for example when the rider signs in again.
func (r *AuthRepository) CancelAccountDeletion(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx, `UPDATE users SET deletion_scheduled_at = NULL WHERE id = $1`, userID)
	return err
}

// HasOpenTheftReport reports whether the user has a theft report that the bike has not been found for.
func (r *AuthRepository) HasOpenTheftReport(ctx context.Context, userID string) (bool, error) {
	var open bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM theft_incidents WHERE user_id = $1 AND resolved_at IS NULL)`,
		userID).Scan(&open)
	return open, err
}

// ListAccountsDueForDeletion returns scheduled accounts whose grace period ended before `before`.
// Accounts with an open theft report are kept, so evidence is not removed.
func (r *AuthRepository) ListAccountsDueForDeletion(ctx context.Context, before time.Time) ([]string, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT u.id FROM users u
		WHERE u.deletion_scheduled_at IS NOT NULL
		  AND u.deletion_scheduled_at < $1
		  AND NOT EXISTS (SELECT 1 FROM theft_incidents i WHERE i.user_id = u.id::text AND i.resolved_at IS NULL)
		ORDER BY u.deletion_scheduled_at
	`, before)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
