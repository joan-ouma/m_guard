package notification

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNoPool      = errors.New("database pool is not initialized")
	ErrNoEndpoints = errors.New("no registered notification endpoints for user")
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) SaveEndpoint(ctx context.Context, ep *DeviceEndpoint) error {
	if r.pool == nil {
		return ErrNoPool
	}
	if ep.ID == "" {
		ep.ID = uuid.NewString()
	}

	now := time.Now().UTC()
	ep.CreatedAt = now
	ep.UpdatedAt = now

	query := `
		INSERT INTO user_notification_endpoints (id, user_id, endpoint_arn, device_token, platform, phone_number, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (user_id, device_token)
		DO UPDATE SET
			endpoint_arn = EXCLUDED.endpoint_arn,
			platform = EXCLUDED.platform,
			phone_number = COALESCE(NULLIF(EXCLUDED.phone_number, ''), user_notification_endpoints.phone_number),
			updated_at = EXCLUDED.updated_at
	`
	_, err := r.pool.Exec(ctx, query, ep.ID, ep.UserID, ep.EndpointARN, ep.DeviceToken, ep.Platform, ep.PhoneNumber, ep.CreatedAt, ep.UpdatedAt)
	return err
}

func (r *Repository) GetEndpointsByUserID(ctx context.Context, userID string) ([]DeviceEndpoint, error) {
	if r.pool == nil {
		return nil, ErrNoPool
	}

	query := `
		SELECT id, user_id, endpoint_arn, device_token, platform, COALESCE(phone_number, ''), created_at, updated_at
		FROM user_notification_endpoints
		WHERE user_id = $1
	`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var endpoints []DeviceEndpoint
	for rows.Next() {
		var ep DeviceEndpoint
		err := rows.Scan(&ep.ID, &ep.UserID, &ep.EndpointARN, &ep.DeviceToken, &ep.Platform, &ep.PhoneNumber, &ep.CreatedAt, &ep.UpdatedAt)
		if err != nil {
			continue
		}
		endpoints = append(endpoints, ep)
	}

	return endpoints, nil
}

func (r *Repository) DeleteEndpointByARN(ctx context.Context, endpointARN string) error {
	if r.pool == nil {
		return ErrNoPool
	}

	query := `DELETE FROM user_notification_endpoints WHERE endpoint_arn = $1`
	_, err := r.pool.Exec(ctx, query, endpointARN)
	return err
}

func (r *Repository) DeleteEndpointByUser(ctx context.Context, userID string) error {
	if r.pool == nil {
		return ErrNoPool
	}

	query := `DELETE FROM user_notification_endpoints WHERE user_id = $1`
	_, err := r.pool.Exec(ctx, query, userID)
	return err
}
