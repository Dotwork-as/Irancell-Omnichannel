package database

import (
	"context"
	"path/filepath"
	"testing"
)

func testMigrations() []Migration {
	return []Migration{
		{ID: 1, Name: "create_alpha", SQL: "CREATE TABLE alpha (id INTEGER PRIMARY KEY)"},
		{ID: 2, Name: "create_beta", SQL: "CREATE TABLE beta (id INTEGER PRIMARY KEY, alpha_id INTEGER REFERENCES alpha(id))"},
	}
}

func tableExists(t *testing.T, db *DB, name string) bool {
	t.Helper()
	var n int
	err := db.QueryRowContext(context.Background(),
		"SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?", name).Scan(&n)
	if err != nil {
		t.Fatalf("sqlite_master lookup for %s: %v", name, err)
	}
	return n == 1
}

func TestMigrateAppliesAllAndIsIdempotent(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	for run := 1; run <= 2; run++ {
		if err := db.Migrate(context.Background(), testMigrations()); err != nil {
			t.Fatalf("Migrate run %d: %v", run, err)
		}
	}

	if !tableExists(t, db, "alpha") || !tableExists(t, db, "beta") {
		t.Error("expected alpha and beta tables to exist after Migrate")
	}

	var recorded int
	if err := db.QueryRowContext(context.Background(),
		"SELECT COUNT(*) FROM schema_migrations").Scan(&recorded); err != nil {
		t.Fatalf("count migrations: %v", err)
	}
	if recorded != 2 {
		t.Errorf("recorded migrations = %d, want 2 (no duplicates on re-run)", recorded)
	}
}

func TestMigrateFailureRollsBackThatMigrationOnly(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	migrations := testMigrations()
	migrations = append(migrations, Migration{
		ID:   3,
		Name: "broken",
		SQL:  "CREATE TABLE broken (this is not valid sql",
	})

	if err := db.Migrate(context.Background(), migrations); err == nil {
		t.Fatal("Migrate with broken SQL: expected error, got nil")
	}

	// Migrations 1 and 2 landed; 3 left no table and no record.
	if !tableExists(t, db, "alpha") || !tableExists(t, db, "beta") {
		t.Error("earlier migrations should remain applied")
	}
	if tableExists(t, db, "broken") {
		t.Error("broken migration must not leave its table behind")
	}

	var recorded int
	if err := db.QueryRowContext(context.Background(),
		"SELECT COUNT(*) FROM schema_migrations").Scan(&recorded); err != nil {
		t.Fatalf("count migrations: %v", err)
	}
	if recorded != 2 {
		t.Errorf("recorded migrations = %d, want 2", recorded)
	}
}

func TestMigrateRejectsInvalidLists(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	cases := map[string][]Migration{
		"duplicate id": {
			{ID: 1, Name: "a", SQL: "SELECT 1"},
			{ID: 1, Name: "b", SQL: "SELECT 1"},
		},
		"non-positive id": {
			{ID: 0, Name: "a", SQL: "SELECT 1"},
		},
		"empty name": {
			{ID: 1, Name: "", SQL: "SELECT 1"},
		},
	}
	for name, ms := range cases {
		if err := db.Migrate(context.Background(), ms); err == nil {
			t.Errorf("%s: expected error, got nil", name)
		}
	}
}

func TestMigrateAppliesUnsortedListInIDOrder(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// beta references alpha, so alpha (ID 1) must apply first even
	// though it is listed second.
	ms := []Migration{
		{ID: 2, Name: "create_beta", SQL: "CREATE TABLE beta (alpha_id INTEGER REFERENCES alpha(id))"},
		{ID: 1, Name: "create_alpha", SQL: "CREATE TABLE alpha (id INTEGER PRIMARY KEY)"},
	}
	if err := db.Migrate(context.Background(), ms); err != nil {
		t.Fatalf("Migrate unsorted: %v", err)
	}
	if !tableExists(t, db, "alpha") || !tableExists(t, db, "beta") {
		t.Error("expected both tables after ordered apply")
	}
}

// The real schema history must apply cleanly and
// idempotently to a fresh database.
func TestMigrationsApplyOnFreshDatabase(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	for run := 1; run <= 2; run++ {
		if err := db.Migrate(context.Background(), Migrations); err != nil {
			t.Fatalf("Migrate run %d: %v", run, err)
		}
	}

	wantTables := []string{
		"products", "product_variants", "product_specs", "product_reviews",
		"shipping_methods", "guest_sessions", "carts", "cart_items",
		"orders", "order_items",
	}
	for _, name := range wantTables {
		if !tableExists(t, db, name) {
			t.Errorf("table %s missing after Migrations", name)
		}
	}

	var recorded int
	if err := db.QueryRowContext(context.Background(),
		"SELECT COUNT(*) FROM schema_migrations").Scan(&recorded); err != nil {
		t.Fatalf("count migrations: %v", err)
	}
	if recorded != len(Migrations) {
		t.Errorf("recorded migrations = %d, want %d", recorded, len(Migrations))
	}
}
