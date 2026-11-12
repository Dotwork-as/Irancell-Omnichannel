// Package database owns the SQLite connection: opening the database
// file with the connection pragmas (WAL, 5s busy timeout, foreign
// keys), and the BEGIN IMMEDIATE transaction helper every multi-statement
// write must use (docs/API.md §6).
package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	// Pure-Go SQLite driver — no CGO toolchain required.
	_ "modernc.org/sqlite"
)

// DB wraps *sql.DB with the SQLite-specific helpers of the backend.
type DB struct {
	*sql.DB
}

// Open opens (creating when absent) the SQLite database at path and
// applies the connection pragmas on every connection: WAL journaling,
// a 5s busy timeout and foreign-key enforcement. The pool is capped
// at exactly one connection — SQLite has a single writer, and WithTx
// relies on that single connection to keep its transaction on one
// session.
func Open(path string) (*DB, error) {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("database: create data directory: %w", err)
		}
	}

	dsn := "file:" + path +
		"?_pragma=busy_timeout(5000)" +
		"&_pragma=journal_mode(WAL)" +
		"&_pragma=foreign_keys(1)"

	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("database: open %s: %w", path, err)
	}
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	if err := sqlDB.Ping(); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("database: ping %s: %w", path, err)
	}
	return &DB{sqlDB}, nil
}

// Querier is the database surface available inside WithTx.
type Querier interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// WithTx runs fn inside one BEGIN IMMEDIATE transaction and commits
// when fn returns nil, rolling back on error or panic. The single
// pool connection is pinned for the whole transaction: concurrent
// writers therefore queue on the pool (clean FIFO-ish wait) instead
// of racing on SQLite's write lock — BEGIN IMMEDIATE can then never
// surface SQLITE_BUSY from a same-process writer. fn must close any
// *sql.Rows it opens before returning.
func (db *DB) WithTx(ctx context.Context, fn func(tx Querier) error) error {
	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("database: acquire connection: %w", err)
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return fmt.Errorf("database: begin immediate: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_, _ = conn.ExecContext(context.WithoutCancel(ctx), "ROLLBACK")
		}
	}()

	if err := fn(conn); err != nil {
		return err
	}
	if _, err := conn.ExecContext(ctx, "COMMIT"); err != nil {
		return fmt.Errorf("database: commit: %w", err)
	}
	committed = true
	return nil
}
