package library

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"testing"
)

func TestOpenCreatesSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "library.db")
	l, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer l.Close()

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("Stat: %v", err)
	}

	v, err := l.SchemaVersion(context.Background())
	if err != nil {
		t.Fatalf("SchemaVersion: %v", err)
	}
	if v != 1 {
		t.Errorf("SchemaVersion = %d, want 1", v)
	}

	rows, err := l.db.Query("SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%' ORDER BY name")
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("Scan: %v", err)
		}
		names = append(names, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows.Err: %v", err)
	}
	want := []string{"cache_meta", "items", "likes", "ratings", "watched", "watchlist"}
	if !reflect.DeepEqual(names, want) {
		t.Errorf("tables = %v, want %v", names, want)
	}
}

func TestOpenPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permissions are not meaningful on windows")
	}
	path := filepath.Join(t.TempDir(), "sub", "library.db")
	l, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer l.Close()

	dirInfo, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatalf("Stat dir: %v", err)
	}
	if got := dirInfo.Mode().Perm(); got != 0o700 {
		t.Errorf("dir perm = %o, want %o", got, 0o700)
	}
	fileInfo, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat file: %v", err)
	}
	if got := fileInfo.Mode().Perm(); got != 0o600 {
		t.Errorf("file perm = %o, want %o", got, 0o600)
	}
}

func TestOpenForeignKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "library.db")
	l, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer l.Close()

	var fk int
	if err := l.db.QueryRow("PRAGMA foreign_keys").Scan(&fk); err != nil {
		t.Fatalf("QueryRow: %v", err)
	}
	if fk != 1 {
		t.Errorf("PRAGMA foreign_keys = %d, want 1", fk)
	}
}

func TestOpenReopen(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "library.db")

	l, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := l.SetMeta(ctx, "k", "v"); err != nil {
		t.Fatalf("SetMeta: %v", err)
	}
	if err := l.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	reopened, err := Open(path)
	if err != nil {
		t.Fatalf("Open second: %v", err)
	}
	defer reopened.Close()

	v, err := reopened.SchemaVersion(ctx)
	if err != nil {
		t.Fatalf("SchemaVersion: %v", err)
	}
	if v != 1 {
		t.Errorf("SchemaVersion = %d, want 1", v)
	}
	got, ok, err := reopened.GetMeta(ctx, "k")
	if err != nil {
		t.Fatalf("GetMeta: %v", err)
	}
	if !ok || got != "v" {
		t.Errorf("GetMeta = %q, %v; want %q, true", got, ok, "v")
	}
}

func TestOpenQuestionMarkInPath(t *testing.T) {
	ctx := context.Background()
	parent := t.TempDir()
	dir := filepath.Join(parent, "q?mark")
	path := filepath.Join(dir, "library.db")

	l, err := Open(path)
	if err != nil {
		if strings.Contains(err.Error(), "must not contain") {
			return
		}
		t.Fatalf("Open: %v", err)
	}
	if err := l.SetMeta(ctx, "k", "v"); err != nil {
		t.Fatalf("SetMeta: %v", err)
	}
	if err := l.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	reopened, err := Open(path)
	if err != nil {
		t.Fatalf("Open second: %v", err)
	}
	defer reopened.Close()
	got, ok, err := reopened.GetMeta(ctx, "k")
	if err != nil {
		t.Fatalf("GetMeta: %v", err)
	}
	if !ok || got != "v" {
		t.Errorf("GetMeta = %q, %v; want %q, true", got, ok, "v")
	}

	// SQLite must not have created the stray file "q" next to the directory.
	if _, err := os.Stat(filepath.Join(parent, "q")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("stray file q: err = %v, want os.ErrNotExist", err)
	}
}

func TestMetaUpsert(t *testing.T) {
	ctx := context.Background()
	l, err := Open(filepath.Join(t.TempDir(), "library.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer l.Close()

	if err := l.SetMeta(ctx, "k", "v1"); err != nil {
		t.Fatalf("SetMeta v1: %v", err)
	}
	if err := l.SetMeta(ctx, "k", "v2"); err != nil {
		t.Fatalf("SetMeta v2: %v", err)
	}
	got, ok, err := l.GetMeta(ctx, "k")
	if err != nil {
		t.Fatalf("GetMeta: %v", err)
	}
	if !ok || got != "v2" {
		t.Errorf("GetMeta = %q, %v; want %q, true", got, ok, "v2")
	}

	got, ok, err = l.GetMeta(ctx, "missing")
	if err != nil {
		t.Fatalf("GetMeta missing: %v", err)
	}
	if ok || got != "" {
		t.Errorf("GetMeta missing = %q, %v; want %q, false", got, ok, "")
	}
}

func TestOpenRefusesNewerSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "library.db")
	l, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if _, err := l.db.Exec("PRAGMA user_version = 99"); err != nil {
		t.Fatalf("Exec: %v", err)
	}
	if err := l.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	_, err = Open(path)
	if err == nil {
		t.Fatal("Open = nil, want error")
	}
	if !strings.Contains(err.Error(), "newer than this build supports") {
		t.Errorf("error = %q, want it to contain %q", err, "newer than this build supports")
	}
}

func TestMigrateRollsBackFailedStep(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "m.db")
	db, err := sql.Open("sqlite", path+"?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	steps := []string{
		"CREATE TABLE a (x INTEGER);",
		"CREATE TABLE b (y INTEGER); THIS IS NOT SQL;",
	}
	err = migrate(ctx, db, steps)
	if err == nil {
		t.Fatal("migrate = nil, want error")
	}
	if !strings.Contains(err.Error(), "migration to version 2") {
		t.Errorf("error = %q, want it to contain %q", err, "migration to version 2")
	}

	v, err := schemaVersion(ctx, db)
	if err != nil {
		t.Fatalf("schemaVersion: %v", err)
	}
	if v != 1 {
		t.Errorf("user_version = %d, want 1", v)
	}

	var name string
	if err := db.QueryRow("SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'a'").Scan(&name); err != nil {
		t.Errorf("table a: %v", err)
	}
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'b'").Scan(&name)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("table b lookup = %v, want sql.ErrNoRows", err)
	}
}

func TestForeignKeyRejectsOrphanLike(t *testing.T) {
	l, err := Open(filepath.Join(t.TempDir(), "library.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer l.Close()

	if _, err := l.db.Exec("INSERT INTO likes VALUES ('movie', 1, 1)"); err == nil {
		t.Fatal("insert = nil, want error")
	}
}

// Several cinedeck processes can start at once on a fresh database. Each
// goroutine opens its own Library, like a separate process would.
func TestConcurrentOpenAndWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "library.db")
	const n = 8
	errs := make(chan error, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			l, err := Open(path)
			if err != nil {
				errs <- err
				return
			}
			defer l.Close()
			ctx := context.Background()
			m := inception()
			if err := l.UpsertItem(ctx, m); err != nil {
				errs <- err
				return
			}
			if _, err := l.Rate(ctx, m.ID, 9); err != nil {
				errs <- err
				return
			}
			if _, err := l.Like(ctx, m.ID); err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("concurrent use: %v", err)
	}
}
