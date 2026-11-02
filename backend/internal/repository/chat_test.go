package repository

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Runs only when TEST_DATABASE_URL points at a disposable database that already has the chat tables.
func TestTakeAIUsageRespectsLimit(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	repo := NewChatRepository(pool)
	day := "2099-01-01"
	_, _ = pool.Exec(ctx, `DELETE FROM chat_ai_usage WHERE day = $1`, day)
	defer pool.Exec(ctx, `DELETE FROM chat_ai_usage WHERE day = $1`, day)

	for i, want := range []bool{true, true, false} {
		got, err := repo.TakeAIUsage(ctx, day, 2)
		if err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
		if got != want {
			t.Fatalf("call %d: got %v, want %v", i, got, want)
		}
	}
	if ok, _ := repo.TakeAIUsage(ctx, day, 0); ok {
		t.Fatal("a zero limit should allow nothing")
	}
}
