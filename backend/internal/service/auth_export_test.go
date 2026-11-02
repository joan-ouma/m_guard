package service

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"motoguard-backend/internal/repository"
	"motoguard-backend/internal/security"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Runs only when TEST_DATABASE_URL points at a disposable database with all migrations applied.
func TestExportUserDataContainsOnlyTheRidersOwnData(t *testing.T) {
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
		me         = "00000000-0000-4000-8000-0000000000f1"
		other      = "00000000-0000-4000-8000-0000000000f2"
		myBike     = "00000000-0000-4000-8000-0000000000c1"
		otherBike  = "00000000-0000-4000-8000-0000000000c2"
		myEmail    = "export-me@example.com"
		otherEmail = "export-other@example.com"
	)
	hash, err := security.HashPassword("export-test-password", nil)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}

	cleanup := func() {
		for _, stmt := range []string{
			`DELETE FROM chat_messages WHERE user_id IN ($1, $2)`,
			`DELETE FROM theft_incidents WHERE user_id IN ($1, $2)`,
			`DELETE FROM user_contacts WHERE user_id IN ($1, $2)`,
			`DELETE FROM driver_contacts WHERE user_id IN ($1, $2)`,
			`DELETE FROM users WHERE id IN ($1, $2)`,
		} {
			_, _ = pool.Exec(ctx, stmt, me, other)
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
	mustExec(`INSERT INTO users (id, email, password_hash, name) VALUES ($1, $3, $5, 'Export Rider'), ($2, $4, $5, 'Other Rider')`, me, other, myEmail, otherEmail, hash)
	mustExec(`INSERT INTO bikes (id, user_id, registration_number, imei) VALUES ($1, $3, 'KEX 111A', '862000000008111'), ($2, $4, 'KEX 222B', '862000000008222')`, myBike, otherBike, me, other)
	mustExec(`INSERT INTO bike_breadcrumbs (bike_id, latitude, longitude) VALUES ($1, -1.28, 36.82), ($2, -1.29, 36.83)`, myBike, otherBike)
	mustExec(`INSERT INTO chat_messages (user_id, role, text) VALUES ($1, 'bot', 'my answer'), ($2, 'bot', 'their answer')`, me, other)
	mustExec(`INSERT INTO theft_incidents (id, user_id, bike_id, mode, lost_at, location_lost) VALUES ('inc_export_test_1', $1, $2, 'other', NOW(), 'My gate')`, me, myBike)
	mustExec(`INSERT INTO user_contacts (id, user_id, name, phone, category) VALUES ('con_export_test_1', $1, 'My contact', '0700000011', 'emergency'), ('con_export_test_2', $2, 'Their contact', '0700000022', 'emergency')`, me, other)
	mustExec(`INSERT INTO driver_contacts (user_id, name, phone) VALUES ($1, 'My driver', '0700000033'), ($2, 'Their driver', '0700000044')`, me, other)

	svc := NewAuthService(pool, repository.NewAuthRepository(pool), security.NewJWTManager("test-secret-key-with-at-least-32-bytes!!", 0), nil, nil, "")
	export, err := svc.ExportUserData(ctx, me)
	if err != nil {
		t.Fatalf("export failed: %v", err)
	}
	raw, err := json.Marshal(export)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	text := string(raw)

	for _, want := range []string{myEmail, "KEX 111A", "my answer", "My gate", "My contact", "My driver"} {
		if !strings.Contains(text, want) {
			t.Errorf("export is missing %q", want)
		}
	}
	for _, unwanted := range []string{otherEmail, "KEX 222B", "their answer", "Their contact", "Their driver", hash, "password_hash"} {
		if strings.Contains(text, unwanted) {
			t.Errorf("export leaks %q", unwanted)
		}
	}
}
