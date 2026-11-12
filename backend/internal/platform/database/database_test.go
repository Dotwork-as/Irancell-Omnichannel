package database

import (
	"context"
	"path/filepath"
	"testing"
)

func openTestDB(t *testing.T) *DB {
	t.Helper()
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestOpenCreatesDatabaseAndDirectory(t *testing.T) {
	db := openTestDB(t)

	var one int
	err := db.QueryRowContext(context.Background(), "SELECT 1").Scan(&one)
	if err != nil || one != 1 {
		t.Fatalf("SELECT 1 = %d, err %v; want 1, nil", one, err)
	}
}

func TestOpenAppliesPragmas(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	var journal string
	if err := db.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&journal); err != nil {
		t.Fatalf("journal_mode: %v", err)
	}
	if journal != "wal" {
		t.Errorf("journal_mode = %q, want wal", journal)
	}

	var fk int
	if err := db.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&fk); err != nil {
		t.Fatalf("foreign_keys: %v", err)
	}
	if fk != 1 {
		t.Errorf("foreign_keys = %d, want 1", fk)
	}

	var busy int
	if err := db.QueryRowContext(ctx, "PRAGMA busy_timeout").Scan(&busy); err != nil {
		t.Fatalf("busy_timeout: %v", err)
	}
	if busy != 5000 {
		t.Errorf("busy_timeout = %d, want 5000", busy)
	}
}

func TestWithTxCommits(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, "CREATE TABLE t (id INTEGER PRIMARY KEY)"); err != nil {
		t.Fatalf("create table: %v", err)
	}

	err := db.WithTx(ctx, func(tx Querier) error {
		_, err := tx.ExecContext(ctx, "INSERT INTO t (id) VALUES (1)")
		return err
	})
	if err != nil {
		t.Fatalf("WithTx: %v", err)
	}

	var n int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM t").Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 1 {
		t.Errorf("rows after commit = %d, want 1", n)
	}
}

func TestWithTxRollsBackOnError(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, "CREATE TABLE t (id INTEGER PRIMARY KEY)"); err != nil {
		t.Fatalf("create table: %v", err)
	}

	err := db.WithTx(ctx, func(tx Querier) error {
		if _, err := tx.ExecContext(ctx, "INSERT INTO t (id) VALUES (1)"); err != nil {
			return err
		}
		return context.DeadlineExceeded // simulate a failing step after the write
	})
	if err == nil {
		t.Fatal("WithTx: expected error, got nil")
	}

	var n int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM t").Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 0 {
		t.Errorf("rows after rollback = %d, want 0", n)
	}
}

func TestWithTxRollsBackOnPanicAndStaysUsable(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, "CREATE TABLE t (id INTEGER PRIMARY KEY)"); err != nil {
		t.Fatalf("create table: %v", err)
	}

	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected panic to propagate out of WithTx fn")
			}
		}()
		_ = db.WithTx(ctx, func(tx Querier) error {
			if _, err := tx.ExecContext(ctx, "INSERT INTO t (id) VALUES (1)"); err != nil {
				return err
			}
			panic("boom")
		})
	}()

	var n int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM t").Scan(&n); err != nil {
		t.Fatalf("count after panic rollback: %v", err)
	}
	if n != 0 {
		t.Errorf("rows after panic = %d, want 0", n)
	}

	// The connection must be usable again after the rollback.
	if err := db.WithTx(ctx, func(tx Querier) error {
		_, err := tx.ExecContext(ctx, "INSERT INTO t (id) VALUES (2)")
		return err
	}); err != nil {
		t.Fatalf("WithTx after panic: %v", err)
	}
}
