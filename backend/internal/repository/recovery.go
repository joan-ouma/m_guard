package repository

import (
	"context"

	"motoguard-backend/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

type RecoveryRepository struct {
	pool *pgxpool.Pool
}

func NewRecoveryRepository(pool *pgxpool.Pool) *RecoveryRepository {
	return &RecoveryRepository{pool: pool}
}

func (r *RecoveryRepository) BikeOwnedBy(ctx context.Context, userID, bikeID string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM bikes WHERE id = $1 AND user_id = $2)`,
		bikeID, userID,
	).Scan(&exists)
	return exists, err
}

func (r *RecoveryRepository) CreateIncident(ctx context.Context, in domain.TheftIncident) (domain.TheftIncident, error) {
	var out domain.TheftIncident
	err := r.pool.QueryRow(ctx, `
		INSERT INTO theft_incidents (id, user_id, bike_id, mode, lost_at, location_lost, latitude, longitude)
		VALUES ($1, $2, NULLIF($3, ''), $4, $5, $6, $7, $8)
		RETURNING id, user_id, COALESCE(bike_id, ''), mode, lost_at, location_lost,
		          latitude, longitude, COALESCE(ob_number, ''), created_at
	`, in.ID, in.UserID, in.BikeID, in.Mode, in.LostAt, in.LocationLost, in.Latitude, in.Longitude,
	).Scan(&out.ID, &out.UserID, &out.BikeID, &out.Mode, &out.LostAt, &out.LocationLost,
		&out.Latitude, &out.Longitude, &out.OBNumber, &out.CreatedAt)
	return out, err
}

func (r *RecoveryRepository) ListIncidents(ctx context.Context, userID string, limit, offset int) ([]domain.TheftIncident, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT i.id, i.user_id, COALESCE(i.bike_id, ''), i.mode, i.lost_at, i.location_lost,
		       i.latitude, i.longitude, COALESCE(i.ob_number, ''), COALESCE(a.file_name, ''), i.created_at
		FROM theft_incidents i
		LEFT JOIN incident_attachments a ON a.incident_id = i.id
		WHERE i.user_id = $1
		ORDER BY i.created_at DESC, i.id DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.TheftIncident{}
	for rows.Next() {
		var in domain.TheftIncident
		if err := rows.Scan(&in.ID, &in.UserID, &in.BikeID, &in.Mode, &in.LostAt, &in.LocationLost,
			&in.Latitude, &in.Longitude, &in.OBNumber, &in.AttachmentName, &in.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, in)
	}
	return out, rows.Err()
}

func (r *RecoveryRepository) CountIncidents(ctx context.Context, userID string) (int, error) {
	var total int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM theft_incidents WHERE user_id = $1`, userID).Scan(&total)
	return total, err
}

// SetOB stores the OB number and, when given, replaces the attachment. Both writes share one transaction.
func (r *RecoveryRepository) SetOB(ctx context.Context, userID, incidentID, obNumber string, attachment *domain.IncidentAttachment) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `
		UPDATE theft_incidents SET ob_number = $3, updated_at = NOW()
		WHERE id = $1 AND user_id = $2
	`, incidentID, userID, obNumber)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	if attachment != nil {
		_, err = tx.Exec(ctx, `
			INSERT INTO incident_attachments (incident_id, file_name, content_type, size_bytes, content)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (incident_id) DO UPDATE SET
				file_name = EXCLUDED.file_name,
				content_type = EXCLUDED.content_type,
				size_bytes = EXCLUDED.size_bytes,
				content = EXCLUDED.content,
				created_at = NOW()
		`, incidentID, attachment.FileName, attachment.ContentType, len(attachment.Data), attachment.Data)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *RecoveryRepository) DeleteIncident(ctx context.Context, userID, incidentID string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM theft_incidents WHERE id = $1 AND user_id = $2`, incidentID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *RecoveryRepository) CreateContact(ctx context.Context, c domain.UserContact) (domain.UserContact, error) {
	var out domain.UserContact
	err := r.pool.QueryRow(ctx, `
		INSERT INTO user_contacts (id, user_id, name, phone, category)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, user_id, name, phone, category, created_at
	`, c.ID, c.UserID, c.Name, c.Phone, c.Category,
	).Scan(&out.ID, &out.UserID, &out.Name, &out.Phone, &out.Category, &out.CreatedAt)
	return out, err
}

func (r *RecoveryRepository) ListContacts(ctx context.Context, userID string) ([]domain.UserContact, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, user_id, name, phone, category, created_at
		FROM user_contacts
		WHERE user_id = $1
		ORDER BY created_at DESC, id DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.UserContact{}
	for rows.Next() {
		var c domain.UserContact
		if err := rows.Scan(&c.ID, &c.UserID, &c.Name, &c.Phone, &c.Category, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *RecoveryRepository) CountContacts(ctx context.Context, userID string) (int, error) {
	var total int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM user_contacts WHERE user_id = $1`, userID).Scan(&total)
	return total, err
}

func (r *RecoveryRepository) DeleteContact(ctx context.Context, userID, contactID string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM user_contacts WHERE id = $1 AND user_id = $2`, contactID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
