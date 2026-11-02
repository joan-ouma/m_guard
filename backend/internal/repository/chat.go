package repository

import (
	"context"
	"errors"
	"time"

	"motoguard-backend/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ChatRepository struct {
	pool *pgxpool.Pool
}

func NewChatRepository(pool *pgxpool.Pool) *ChatRepository {
	return &ChatRepository{pool: pool}
}

func (r *ChatRepository) Append(ctx context.Context, userID, role, text string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO chat_messages (user_id, role, text) VALUES ($1, $2, $3)`,
		userID, role, text)
	return err
}

// Recent returns the newest `limit` messages, oldest first.
func (r *ChatRepository) Recent(ctx context.Context, userID string, limit int) ([]domain.ChatHistoryItem, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, role, text, created_at
		FROM (
			SELECT id, role, text, created_at
			FROM chat_messages
			WHERE user_id = $1
			ORDER BY created_at DESC, id DESC
			LIMIT $2
		) recent
		ORDER BY created_at ASC, id ASC
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.ChatHistoryItem{}
	for rows.Next() {
		var item domain.ChatHistoryItem
		if err := rows.Scan(&item.ID, &item.Role, &item.Text, &item.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *ChatRepository) Clear(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM chat_messages WHERE user_id = $1`, userID)
	return err
}

// PurgeOlderThan deletes messages created before cutoff and returns how many were removed.
func (r *ChatRepository) PurgeOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	tag, err := r.pool.Exec(ctx, `DELETE FROM chat_messages WHERE created_at < $1`, cutoff)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// TakeAIUsage reserves one OpenAI call for the given day (YYYY-MM-DD) if fewer than limit have been made.
// The check and the increment happen in one statement, so concurrent requests cannot overshoot the limit.
func (r *ChatRepository) TakeAIUsage(ctx context.Context, day string, limit int) (bool, error) {
	if limit <= 0 {
		return false, nil
	}
	var calls int
	err := r.pool.QueryRow(ctx, `
		INSERT INTO chat_ai_usage (day, calls) VALUES ($1, 1)
		ON CONFLICT (day) DO UPDATE SET calls = chat_ai_usage.calls + 1
		WHERE chat_ai_usage.calls < $2
		RETURNING calls
	`, day, limit).Scan(&calls)
	if errors.Is(err, pgx.ErrNoRows) {
		// The day's row already holds the limit, so the update was skipped.
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
