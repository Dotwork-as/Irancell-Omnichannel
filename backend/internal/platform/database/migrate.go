package database

import (
	"context"
	"fmt"
	"sort"
)

// Migration is one schema step. IDs must be unique and positive;
// migrations apply once, in ascending ID order, and are recorded in
// schema_migrations. The schema history itself lives in migrations.go.
type Migration struct {
	ID   int64
	Name string
	SQL  string
}

// Migrate applies every migration not yet recorded, each inside its
// own transaction: a migration either lands completely or not at all,
// and re-running against an up-to-date database is a no-op.
func (db *DB) Migrate(ctx context.Context, migrations []Migration) error {
	if err := validateMigrations(migrations); err != nil {
		return err
	}
	if err := db.ensureMigrationTable(ctx); err != nil {
		return fmt.Errorf("database: migration table: %w", err)
	}

	seen, err := db.appliedMigrations(ctx)
	if err != nil {
		return fmt.Errorf("database: read applied migrations: %w", err)
	}

	list := append([]Migration(nil), migrations...)
	sort.Slice(list, func(i, j int) bool { return list[i].ID < list[j].ID })

	for _, m := range list {
		if seen[m.ID] {
			continue
		}
		err := db.WithTx(ctx, func(tx Querier) error {
			if _, err := tx.ExecContext(ctx, m.SQL); err != nil {
				return fmt.Errorf("migration %d (%s): %w", m.ID, m.Name, err)
			}
			if _, err := tx.ExecContext(ctx,
				"INSERT INTO schema_migrations (id, name) VALUES (?, ?)",
				m.ID, m.Name,
			); err != nil {
				return fmt.Errorf("migration %d (%s): record: %w", m.ID, m.Name, err)
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func validateMigrations(migrations []Migration) error {
	ids := make(map[int64]struct{}, len(migrations))
	for _, m := range migrations {
		if m.ID <= 0 {
			return fmt.Errorf("database: migration %q has non-positive ID %d", m.Name, m.ID)
		}
		if m.Name == "" {
			return fmt.Errorf("database: migration %d has an empty name", m.ID)
		}
		if _, dup := ids[m.ID]; dup {
			return fmt.Errorf("database: duplicate migration ID %d", m.ID)
		}
		ids[m.ID] = struct{}{}
	}
	return nil
}

func (db *DB) ensureMigrationTable(ctx context.Context) error {
	const q = `CREATE TABLE IF NOT EXISTS schema_migrations (
		id         INTEGER PRIMARY KEY,
		name       TEXT NOT NULL,
		applied_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
	)`
	_, err := db.ExecContext(ctx, q)
	return err
}

func (db *DB) appliedMigrations(ctx context.Context) (map[int64]bool, error) {
	rows, err := db.QueryContext(ctx, "SELECT id FROM schema_migrations")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	seen := make(map[int64]bool)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		seen[id] = true
	}
	return seen, rows.Err()
}
