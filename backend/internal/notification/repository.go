package notification

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool       *pgxpool.Pool
	mu         sync.RWMutex
	inMemoryDB map[string][]DeviceEndpoint // Fallback if table does not exist yet
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	repo := &Repository{
		pool:       pool,
		inMemoryDB: make(map[string][]DeviceEndpoint),
	}
	repo.initSchema()
	return repo
}

func (r *Repository) initSchema() {
	if r.pool == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		CREATE TABLE IF NOT EXISTS user_notification_endpoints (
			id VARCHAR(64) PRIMARY KEY,
			user_id VARCHAR(64) NOT NULL,
			endpoint_arn TEXT NOT NULL,
			device_token TEXT NOT NULL,
			platform VARCHAR(32) NOT NULL DEFAULT 'GCM',
			phone_number VARCHAR(32),
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS idx_user_notification_endpoints_user_id ON user_notification_endpoints(user_id);
	`
	_, _ = r.pool.Exec(ctx, query)
}

func (r *Repository) SaveEndpoint(ctx context.Context, ep *DeviceEndpoint) error {
	if ep.ID == "" {
		ep.ID = uuid.NewString()
	}
	now := time.Now().UTC()
	ep.CreatedAt = now
	ep.UpdatedAt = now

	// Save to DB if available
	if r.pool != nil {
		query := `
			INSERT INTO user_notification_endpoints (id, user_id, endpoint_arn, device_token, platform, phone_number, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (id) DO UPDATE SET
				endpoint_arn = EXCLUDED.endpoint_arn,
				device_token = EXCLUDED.device_token,
				platform = EXCLUDED.platform,
				phone_number = EXCLUDED.phone_number,
				updated_at = EXCLUDED.updated_at
		`
		_, err := r.pool.Exec(ctx, query, ep.ID, ep.UserID, ep.EndpointARN, ep.DeviceToken, ep.Platform, ep.PhoneNumber, ep.CreatedAt, ep.UpdatedAt)
		if err == nil {
			return nil
		}
	}

	// In-memory fallback
	r.mu.Lock()
	defer r.mu.Unlock()
	list := r.inMemoryDB[ep.UserID]
	// Replace if device token already exists
	updated := false
	for i, existing := range list {
		if existing.DeviceToken == ep.DeviceToken {
			list[i] = *ep
			updated = true
			break
		}
	}
	if !updated {
		list = append(list, *ep)
	}
	r.inMemoryDB[ep.UserID] = list
	return nil
}

func (r *Repository) GetEndpointsByUserID(ctx context.Context, userID string) ([]DeviceEndpoint, error) {
	if r.pool != nil {
		query := `
			SELECT id, user_id, endpoint_arn, device_token, platform, COALESCE(phone_number, ''), created_at, updated_at
			FROM user_notification_endpoints
			WHERE user_id = $1
		`
		rows, err := r.pool.Query(ctx, query, userID)
		if err == nil {
			defer rows.Close()
			var result []DeviceEndpoint
			for rows.Next() {
				var ep DeviceEndpoint
				if scanErr := rows.Scan(&ep.ID, &ep.UserID, &ep.EndpointARN, &ep.DeviceToken, &ep.Platform, &ep.PhoneNumber, &ep.CreatedAt, &ep.UpdatedAt); scanErr == nil {
					result = append(result, ep)
				}
			}
			if len(result) > 0 {
				return result, nil
			}
		}
	}

	// In-memory fallback
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.inMemoryDB[userID], nil
}

func (r *Repository) DeleteEndpointByARN(ctx context.Context, endpointARN string) error {
	if r.pool != nil {
		query := `DELETE FROM user_notification_endpoints WHERE endpoint_arn = $1`
		_, _ = r.pool.Exec(ctx, query, endpointARN)
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	for uID, list := range r.inMemoryDB {
		var filtered []DeviceEndpoint
		for _, ep := range list {
			if ep.EndpointARN != endpointARN {
				filtered = append(filtered, ep)
			}
		}
		r.inMemoryDB[uID] = filtered
	}
	return nil
}

func (r *Repository) DeleteEndpointByUser(ctx context.Context, userID string) error {
	if r.pool != nil {
		query := `DELETE FROM user_notification_endpoints WHERE user_id = $1`
		_, _ = r.pool.Exec(ctx, query, userID)
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.inMemoryDB, userID)
	return nil
}

var ErrNoEndpoints = errors.New("no registered notification endpoints for user")
var _ = pgx.ErrNoRows
