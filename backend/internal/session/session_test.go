package session

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"testing"
	"time"

	"backend/internal/platform/database"
)

func testService(t *testing.T, cookieSecure bool) *Service {
	t.Helper()
	db, err := database.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := db.Migrate(context.Background(), database.Migrations); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return New(db, cookieSecure)
}

func TestCreateAndResolve(t *testing.T) {
	svc := testService(t, true)
	ctx := context.Background()

	token, id, err := svc.Create(ctx)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if token == "" || id == 0 {
		t.Fatalf("token=%q id=%d, want both non-empty", token, id)
	}

	got, err := svc.Resolve(ctx, token)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got != id {
		t.Errorf("Resolve = %d, want %d", got, id)
	}

	// Only the SHA-256 hash may be stored — verify against the row.
	sum := sha256.Sum256([]byte(token))
	var storedHash string
	if err := svc.db.QueryRowContext(ctx,
		"SELECT token_hash FROM guest_sessions WHERE id = ?", id).Scan(&storedHash); err != nil {
		t.Fatalf("lookup hash: %v", err)
	}
	if storedHash == token || storedHash != hex.EncodeToString(sum[:]) {
		t.Errorf("stored hash %q must be sha256(token), never the token itself", storedHash)
	}
}

func TestTokensAreUnique(t *testing.T) {
	svc := testService(t, true)
	t1, _, err := svc.Create(context.Background())
	if err != nil {
		t.Fatalf("Create 1: %v", err)
	}
	t2, _, err := svc.Create(context.Background())
	if err != nil {
		t.Fatalf("Create 2: %v", err)
	}
	if t1 == t2 {
		t.Error("two sessions produced identical tokens")
	}
}

func TestResolveInvalidAndEmpty(t *testing.T) {
	svc := testService(t, true)
	ctx := context.Background()

	for _, token := range []string{"", "garbage-token"} {
		got, err := svc.Resolve(ctx, token)
		if err != nil {
			t.Fatalf("Resolve(%q): %v", token, err)
		}
		if got != 0 {
			t.Errorf("Resolve(%q) = %d, want 0 (no session)", token, got)
		}
	}
}

func TestResolveExpired(t *testing.T) {
	svc := testService(t, true)
	ctx := context.Background()

	token, id, err := svc.Create(ctx)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Backdate the session instead of sleeping.
	if _, err := svc.db.ExecContext(ctx,
		"UPDATE guest_sessions SET expires_at = ? WHERE id = ?",
		time.Now().UTC().Add(-time.Minute).Format(time.RFC3339), id); err != nil {
		t.Fatalf("backdate: %v", err)
	}

	got, err := svc.Resolve(ctx, token)
	if err != nil {
		t.Fatalf("Resolve expired: %v", err)
	}
	if got != 0 {
		t.Errorf("Resolve(expired) = %d, want 0 (expired means no session)", got)
	}

	// The row stays in place: cleanup is deferred beyond the MVP.
	var n int
	if err := svc.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM guest_sessions").Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 1 {
		t.Errorf("expired session row = %d, want 1 (cleanup deferred)", n)
	}
}

func TestCookieAttributes(t *testing.T) {
	secure := testService(t, true).Cookie("tok")
	if secure.Name != CookieName || secure.Value != "tok" {
		t.Errorf("name/value = %q/%q", secure.Name, secure.Value)
	}
	if !secure.HTTPOnly || !secure.Secure || secure.SameSite != "Lax" || secure.Path != "/" {
		t.Errorf("cookie flags wrong: httponly=%v secure=%v samesite=%q path=%q",
			secure.HTTPOnly, secure.Secure, secure.SameSite, secure.Path)
	}
	if secure.MaxAge != CookieMaxAge {
		t.Errorf("MaxAge = %d, want %d (30 days)", secure.MaxAge, CookieMaxAge)
	}

	insecure := testService(t, false).Cookie("tok")
	if insecure.Secure {
		t.Error("COOKIE_SECURE=false must produce a non-Secure cookie (local dev only)")
	}
}
