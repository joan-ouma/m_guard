package repository

import (
	"context"
	"time"

	"motoguard-backend/internal/domain"
)

// ExportUserData reads everything held for one rider. Every query is scoped to userID.
func (r *AuthRepository) ExportUserData(ctx context.Context, userID string) (*domain.UserExport, error) {
	out := &domain.UserExport{
		ExportedAt:        time.Now().UTC(),
		Bikes:             []domain.ExportBike{},
		LocationHistory:   []domain.ExportLocation{},
		BikeEvents:        []domain.ExportBikeEvent{},
		TheftReports:      []domain.ExportTheftReport{},
		Contacts:          []domain.ExportContact{},
		DriverContacts:    []domain.ExportDriverContact{},
		AssistantMessages: []domain.ExportChatMessage{},
	}

	if err := r.pool.QueryRow(ctx, `
		SELECT id, email, COALESCE(name, ''), COALESCE(phone_number, ''), COALESCE(national_id, ''), created_at
		FROM users WHERE id = $1`, userID).
		Scan(&out.Profile.ID, &out.Profile.Email, &out.Profile.Name, &out.Profile.PhoneNumber, &out.Profile.NationalID, &out.Profile.CreatedAt); err != nil {
		return nil, err
	}

	if err := queryRows(ctx, r, `
		SELECT id, registration_number, COALESCE(label, ''), COALESCE(make, ''), COALESCE(model, ''), COALESCE(year, 0), COALESCE(color, ''), imei, created_at
		FROM bikes WHERE user_id = $1 ORDER BY created_at`, userID,
		func(rows scanner) error {
			var b domain.ExportBike
			if err := rows.Scan(&b.ID, &b.RegistrationNumber, &b.Label, &b.Make, &b.Model, &b.Year, &b.Color, &b.IMEI, &b.CreatedAt); err != nil {
				return err
			}
			out.Bikes = append(out.Bikes, b)
			return nil
		}); err != nil {
		return nil, err
	}

	if err := queryRows(ctx, r, `
		SELECT bb.bike_id, bb.latitude, bb.longitude, bb.speed_kmh, bb.ignition, bb.recorded_at
		FROM bike_breadcrumbs bb JOIN bikes b ON b.id = bb.bike_id
		WHERE b.user_id = $1 AND bb.recorded_at > NOW() - INTERVAL '90 days'
		ORDER BY bb.recorded_at`, userID,
		func(rows scanner) error {
			var l domain.ExportLocation
			if err := rows.Scan(&l.BikeID, &l.Latitude, &l.Longitude, &l.SpeedKMH, &l.Ignition, &l.RecordedAt); err != nil {
				return err
			}
			out.LocationHistory = append(out.LocationHistory, l)
			return nil
		}); err != nil {
		return nil, err
	}

	if err := queryRows(ctx, r, `
		SELECT be.bike_id, be.event_type, be.latitude, be.longitude, be.recorded_at
		FROM bike_events be JOIN bikes b ON b.id = be.bike_id
		WHERE b.user_id = $1 ORDER BY be.recorded_at`, userID,
		func(rows scanner) error {
			var e domain.ExportBikeEvent
			if err := rows.Scan(&e.BikeID, &e.EventType, &e.Latitude, &e.Longitude, &e.RecordedAt); err != nil {
				return err
			}
			out.BikeEvents = append(out.BikeEvents, e)
			return nil
		}); err != nil {
		return nil, err
	}

	if err := queryRows(ctx, r, `
		SELECT id, COALESCE(bike_id, ''), mode, lost_at, location_lost, COALESCE(ob_number, ''), resolved_at, created_at
		FROM theft_incidents WHERE user_id = $1 ORDER BY created_at`, userID,
		func(rows scanner) error {
			var t domain.ExportTheftReport
			if err := rows.Scan(&t.ID, &t.BikeID, &t.Mode, &t.LostAt, &t.LocationLost, &t.OBNumber, &t.ResolvedAt, &t.CreatedAt); err != nil {
				return err
			}
			out.TheftReports = append(out.TheftReports, t)
			return nil
		}); err != nil {
		return nil, err
	}

	if err := queryRows(ctx, r, `
		SELECT id, name, phone, category, created_at FROM user_contacts WHERE user_id = $1 ORDER BY created_at`, userID,
		func(rows scanner) error {
			var c domain.ExportContact
			if err := rows.Scan(&c.ID, &c.Name, &c.Phone, &c.Category, &c.CreatedAt); err != nil {
				return err
			}
			out.Contacts = append(out.Contacts, c)
			return nil
		}); err != nil {
		return nil, err
	}

	if err := queryRows(ctx, r, `
		SELECT name, phone, COALESCE(email, ''), created_at FROM driver_contacts WHERE user_id = $1 ORDER BY created_at`, userID,
		func(rows scanner) error {
			var d domain.ExportDriverContact
			if err := rows.Scan(&d.Name, &d.Phone, &d.Email, &d.CreatedAt); err != nil {
				return err
			}
			out.DriverContacts = append(out.DriverContacts, d)
			return nil
		}); err != nil {
		return nil, err
	}

	if err := queryRows(ctx, r, `
		SELECT role, text, created_at FROM chat_messages WHERE user_id = $1 ORDER BY created_at`, userID,
		func(rows scanner) error {
			var m domain.ExportChatMessage
			if err := rows.Scan(&m.Role, &m.Text, &m.CreatedAt); err != nil {
				return err
			}
			out.AssistantMessages = append(out.AssistantMessages, m)
			return nil
		}); err != nil {
		return nil, err
	}

	return out, nil
}

type scanner interface {
	Scan(dest ...any) error
}

// queryRows runs a query and calls each for every row, closing the rows when done.
func queryRows(ctx context.Context, r *AuthRepository, query string, arg any, each func(scanner) error) error {
	rows, err := r.pool.Query(ctx, query, arg)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		if err := each(rows); err != nil {
			return err
		}
	}
	return rows.Err()
}
