package repository

import (
	"context"
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
		SELECT id, email, password_hash, name, created_at, updated_at
		FROM users
		WHERE LOWER(email) = LOWER($1)
		LIMIT 1
	`
	row := r.pool.QueryRow(ctx, query, email)

	var u domain.User
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (r *AuthRepository) CreateUser(ctx context.Context, u *domain.User) error {
	query := `
		INSERT INTO users (id, email, password_hash, name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.pool.Exec(ctx, query,
		u.ID,
		u.Email,
		u.PasswordHash,
		u.Name,
		u.CreatedAt,
		u.UpdatedAt,
	)
	return err
}

func (r *AuthRepository) GetUserByID(ctx context.Context, userID string) (*domain.User, error) {
	query := `
		SELECT id, email, password_hash, name, created_at, updated_at
		FROM users
		WHERE id = $1
		LIMIT 1
	`
	row := r.pool.QueryRow(ctx, query, userID)

	var u domain.User
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (r *AuthRepository) GetSessionWithUser(ctx context.Context, tokenHash string) (*domain.Session, *domain.User, error) {
	query := `
		SELECT 
			s.id, s.user_id, s.refresh_token_hash, s.expires_at, COALESCE(s.revoked_at, '0001-01-01 00:00:00+00'), s.user_agent, s.ip_address, s.created_at,
			u.id, u.email, u.password_hash, u.name, u.created_at, u.updated_at
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.refresh_token_hash = $1
		LIMIT 1
	`
	row := r.pool.QueryRow(ctx, query, tokenHash)

	var s domain.Session
	var revokedAt time.Time
	var u domain.User

	err := row.Scan(
		&s.ID, &s.UserID, &s.RefreshTokenHash, &s.ExpiresAt, &revokedAt, &s.UserAgent, &s.IPAddress, &s.CreatedAt,
		&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.CreatedAt, &u.UpdatedAt,
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
