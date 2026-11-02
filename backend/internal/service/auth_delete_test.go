package service

import (
	"context"
	"os"
	"testing"

	"motoguard-backend/internal/repository"
	"motoguard-backend/internal/security"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Runs only when TEST_DATABASE_URL points at a disposable database with all migrations applied.
func TestDeleteAccountRemovesOnlyThatUsersData(t *testing.T) {
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

	const (
		password     = "correct-horse-battery"
		doomed       = "00000000-0000-4000-8000-0000000000d1"
		keeper       = "00000000-0000-4000-8000-0000000000e2"
		doomedBike   = "00000000-0000-4000-8000-0000000000b1"
		keeperBike   = "00000000-0000-4000-8000-0000000000b2"
		doomedIncid  = "inc_delete_test_1"
		doomedContct = "con_delete_test_1"
		keeperContct = "con_delete_test_2"
	)
	hash, err := security.HashPassword(password, nil)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}

	// Remove leftovers from an earlier run. Tables without a foreign key are cleared explicitly.
	cleanup := func() {
		for _, stmt := range []string{
			`DELETE FROM chat_messages WHERE user_id IN ($1, $2)`,
			`DELETE FROM theft_incidents WHERE user_id IN ($1, $2)`,
			`DELETE FROM user_contacts WHERE user_id IN ($1, $2)`,
			`DELETE FROM driver_contacts WHERE user_id IN ($1, $2)`,
			`DELETE FROM users WHERE id IN ($1, $2)`,
		} {
			_, _ = pool.Exec(ctx, stmt, doomed, keeper)
		}
	}
	cleanup()
	defer cleanup()

	mustExec := func(stmt string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, stmt, args...); err != nil {
			t.Fatalf("seed failed: %v\n%s", err, stmt)
		}
	}
	mustExec(`INSERT INTO users (id, email, password_hash) VALUES ($1, 'delete-me@example.com', $3), ($2, 'keep-me@example.com', $3)`, doomed, keeper, hash)
	mustExec(`INSERT INTO bikes (id, user_id, registration_number, imei) VALUES ($1, $3, 'KDT 001A', '862000000009001'), ($2, $4, 'KDT 002B', '862000000009002')`, doomedBike, keeperBike, doomed, keeper)
	mustExec(`INSERT INTO bike_breadcrumbs (bike_id, latitude, longitude) VALUES ($1, -1.28, 36.82), ($2, -1.29, 36.83)`, doomedBike, keeperBike)
	mustExec(`INSERT INTO chat_messages (user_id, role, text) VALUES ($1, 'bot', 'mine'), ($2, 'bot', 'theirs')`, doomed, keeper)
	mustExec(`INSERT INTO theft_incidents (id, user_id, mode, lost_at, location_lost) VALUES ($1, $2, 'other', NOW(), 'Gate 1')`, doomedIncid, doomed)
	mustExec(`INSERT INTO user_contacts (id, user_id, name, phone, category) VALUES ($1, $2, 'Mine', '0700000001', 'emergency'), ($3, $4, 'Theirs', '0700000002', 'emergency')`, doomedContct, doomed, keeperContct, keeper)
	mustExec(`INSERT INTO driver_contacts (user_id, name, phone) VALUES ($1, 'Mine', '0700000003'), ($2, 'Theirs', '0700000004')`, doomed, keeper)

	svc := NewAuthService(pool, repository.NewAuthRepository(pool), security.NewJWTManager("test-secret-key-with-at-least-32-bytes!!", 0), nil, nil, "")

	if err := svc.DeleteAccount(ctx, doomed, "wrong-password"); err == nil {
		t.Fatal("a wrong password must not delete the account")
	}
	if n := count(t, pool, `SELECT count(*) FROM users WHERE id = $1`, doomed); n != 1 {
		t.Fatal("account was deleted despite the wrong password")
	}

	if err := svc.DeleteAccount(ctx, doomed, password); err != nil {
		t.Fatalf("delete with the correct password failed: %v", err)
	}

	// Everything belonging to the deleted account must be gone.
	gone := map[string]int{
		`SELECT count(*) FROM users WHERE id = $1`:                 count(t, pool, `SELECT count(*) FROM users WHERE id = $1`, doomed),
		`SELECT count(*) FROM bikes WHERE user_id = $1`:            count(t, pool, `SELECT count(*) FROM bikes WHERE user_id = $1`, doomed),
		`SELECT count(*) FROM bike_breadcrumbs WHERE bike_id = $1`: count(t, pool, `SELECT count(*) FROM bike_breadcrumbs WHERE bike_id = $1`, doomedBike),
		`SELECT count(*) FROM chat_messages WHERE user_id = $1`:    count(t, pool, `SELECT count(*) FROM chat_messages WHERE user_id = $1`, doomed),
		`SELECT count(*) FROM theft_incidents WHERE user_id = $1`:  count(t, pool, `SELECT count(*) FROM theft_incidents WHERE user_id = $1`, doomed),
		`SELECT count(*) FROM user_contacts WHERE user_id = $1`:    count(t, pool, `SELECT count(*) FROM user_contacts WHERE user_id = $1`, doomed),
		`SELECT count(*) FROM driver_contacts WHERE user_id = $1`:  count(t, pool, `SELECT count(*) FROM driver_contacts WHERE user_id = $1`, doomed),
	}
	for query, n := range gone {
		if n != 0 {
			t.Errorf("%d row(s) remain for the deleted account: %s", n, query)
		}
	}

	// Another rider's data must be untouched.
	kept := map[string]int{
		"account":        count(t, pool, `SELECT count(*) FROM users WHERE id = $1`, keeper),
		"bike":           count(t, pool, `SELECT count(*) FROM bikes WHERE user_id = $1`, keeper),
		"breadcrumbs":    count(t, pool, `SELECT count(*) FROM bike_breadcrumbs WHERE bike_id = $1`, keeperBike),
		"chat history":   count(t, pool, `SELECT count(*) FROM chat_messages WHERE user_id = $1`, keeper),
		"contacts":       count(t, pool, `SELECT count(*) FROM user_contacts WHERE user_id = $1`, keeper),
		"driver contact": count(t, pool, `SELECT count(*) FROM driver_contacts WHERE user_id = $1`, keeper),
	}
	for what, n := range kept {
		if n != 1 {
			t.Errorf("another rider's %s was affected (count %d)", what, n)
		}
	}
}

func count(t *testing.T, pool *pgxpool.Pool, query string, arg any) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), query, arg).Scan(&n); err != nil {
		t.Fatalf("count query failed: %v\n%s", err, query)
	}
	return n
}
