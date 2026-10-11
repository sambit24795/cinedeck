package library

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite" // registers the "sqlite" driver
)

// Library is the local database of items, likes, ratings, watched titles and the watchlist.
type Library struct {
	db  *sql.DB
	now func() time.Time // time.Now; replaced by SetClock in tests
}

// Open opens (creating it if needed) the database at path and migrates it to the latest schema.
func Open(path string) (*Library, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("library: %w", err)
	}

	// SQLite would otherwise create the file with the umask default, which
	// can be world-readable, so create it and tighten the mode explicitly.
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("library: %w", err)
	}
	f.Close()
	if err := os.Chmod(path, 0o600); err != nil {
		return nil, fmt.Errorf("library: %w", err)
	}

	db, err := sql.Open("sqlite", path+"?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("library: %w", err)
	}
	db.SetMaxOpenConns(1)

	if err := migrate(context.Background(), db, migrations); err != nil {
		db.Close()
		return nil, err
	}
	return &Library{db: db, now: time.Now}, nil
}

// Close closes the underlying database.
func (l *Library) Close() error {
	return l.db.Close()
}

// SetClock replaces the time source. It exists for tests.
func (l *Library) SetClock(now func() time.Time) {
	l.now = now
}

// SchemaVersion returns PRAGMA user_version.
func (l *Library) SchemaVersion(ctx context.Context) (int, error) {
	return schemaVersion(ctx, l.db)
}

// SetMeta upserts a key/value pair into cache_meta.
func (l *Library) SetMeta(ctx context.Context, key, value string) error {
	_, err := l.db.ExecContext(ctx,
		`INSERT INTO cache_meta (key, value, updated_at) VALUES (?, ?, ?)
         ON CONFLICT (key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`,
		key, value, l.now().Unix())
	if err != nil {
		return fmt.Errorf("library: %w", err)
	}
	return nil
}

// GetMeta returns the value stored for key. The bool is false when the key is missing.
func (l *Library) GetMeta(ctx context.Context, key string) (string, bool, error) {
	var value string
	err := l.db.QueryRowContext(ctx, "SELECT value FROM cache_meta WHERE key = ?", key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("library: %w", err)
	}
	return value, true, nil
}

// schemaVersion reads PRAGMA user_version from db.
func schemaVersion(ctx context.Context, db *sql.DB) (int, error) {
	var v int
	if err := db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&v); err != nil {
		return 0, fmt.Errorf("library: %w", err)
	}
	return v, nil
}

// migrate applies steps[version:] in order; each step and its version bump are one transaction.
func migrate(ctx context.Context, db *sql.DB, steps []string) error {
	v, err := schemaVersion(ctx, db)
	if err != nil {
		return err
	}
	if v > len(steps) {
		return fmt.Errorf("library: database schema version %d is newer than this build supports (%d); upgrade cinedeck", v, len(steps))
	}

	for i := v; i < len(steps); i++ {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("library: migration to version %d: %w", i+1, err)
		}
		if _, err := tx.ExecContext(ctx, steps[i]); err != nil {
			tx.Rollback()
			return fmt.Errorf("library: migration to version %d: %w", i+1, err)
		}
		if _, err := tx.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version = %d", i+1)); err != nil {
			tx.Rollback()
			return fmt.Errorf("library: migration to version %d: %w", i+1, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("library: migration to version %d: %w", i+1, err)
		}
	}
	return nil
}
