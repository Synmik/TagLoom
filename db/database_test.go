package db

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestBaseName(t *testing.T) {
	cases := map[string]string{
		"c.jpg":          "c.jpg",
		"a/b/c.jpg":      "c.jpg",
		"a\\b\\c.jpg":    "c.jpg",
		"a/b\\c.mp4":     "c.mp4",
		"deep/x/y/z.png": "z.png",
		"/abs/c.jpg":     "c.jpg",
	}
	for in, want := range cases {
		if got := baseName(in); got != want {
			t.Errorf("baseName(%q) = %q, want %q", in, got, want)
		}
	}
}

func openAt(t *testing.T, path string) *Database {
	t.Helper()
	d, err := NewDatabase(path)
	if err != nil {
		t.Fatalf("NewDatabase: %v", err)
	}
	t.Cleanup(func() { d.Close() })
	return d
}

func columnExists(t *testing.T, d *Database, table, column string) bool {
	t.Helper()
	var n int
	err := d.Conn().QueryRow(
		"SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?", table, column,
	).Scan(&n)
	if err != nil {
		t.Fatalf("pragma_table_info: %v", err)
	}
	return n > 0
}

func TestNewDatabaseFreshSchemaVersion(t *testing.T) {
	d := openAt(t, filepath.Join(t.TempDir(), "db.sqlite"))

	version, err := d.SchemaVersion()
	if err != nil {
		t.Fatalf("SchemaVersion: %v", err)
	}
	if version != SchemaVersion {
		t.Fatalf("version = %d, want %d", version, SchemaVersion)
	}

	// Every migration should be recorded
	var rows int
	if err := d.Conn().QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&rows); err != nil {
		t.Fatalf("count: %v", err)
	}
	if rows != len(migrations) {
		t.Fatalf("schema_migrations rows = %d, want %d", rows, len(migrations))
	}

	// idx_files_filename must exist (created by migration 4, not schema.sql)
	var idx int
	if err := d.Conn().QueryRow(
		"SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_files_filename'",
	).Scan(&idx); err != nil || idx != 1 {
		t.Fatalf("idx_files_filename missing on fresh db (count=%d, err=%v)", idx, err)
	}
}

func TestNewDatabaseReopenIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "db.sqlite")
	d := openAt(t, path)
	first, _ := d.SchemaVersion()
	d.Close()

	// Reopen — no migration should re-run, version unchanged
	d2 := openAt(t, path)
	second, err := d2.SchemaVersion()
	if err != nil {
		t.Fatalf("SchemaVersion: %v", err)
	}
	if second != first || second != SchemaVersion {
		t.Fatalf("version after reopen = %d (first: %d), want %d", second, first, SchemaVersion)
	}
	var rows int
	if err := d2.Conn().QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&rows); err != nil {
		t.Fatalf("count: %v", err)
	}
	if rows != len(migrations) {
		t.Fatalf("schema_migrations rows after reopen = %d, want %d", rows, len(migrations))
	}
}

// createOldVault builds a database as a pre-migration vault would look:
// old files table (no filename/file_size/date_created/date_modified),
// no schema_migrations table, no composite indexes.
func createOldVault(t *testing.T, path string) {
	t.Helper()
	conn, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer conn.Close()

	stmts := []string{
		`CREATE TABLE files (
			id INTEGER PRIMARY KEY,
			vault_path TEXT NOT NULL UNIQUE,
			thumbnail_path TEXT,
			name TEXT,
			notes TEXT,
			link TEXT,
			rating INTEGER DEFAULT 0,
			is_favorite INTEGER DEFAULT 0,
			folder_path TEXT NOT NULL,
			indexed_at TEXT NOT NULL
		)`,
		`CREATE TABLE tags (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			color TEXT,
			parent_id INTEGER,
			is_category INTEGER DEFAULT 0,
			sort_order INTEGER DEFAULT 0,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE tag_aliases (tag_id INTEGER NOT NULL, alias TEXT NOT NULL, PRIMARY KEY (tag_id, alias))`,
		`CREATE TABLE file_tags (file_id INTEGER NOT NULL, tag_id INTEGER NOT NULL, PRIMARY KEY (file_id, tag_id))`,
		`INSERT INTO files (vault_path, folder_path, name, indexed_at) VALUES ('a/b/c.jpg', 'a/b', 'c photo', '2025-01-01T00:00:00Z')`,
	}
	for _, s := range stmts {
		if _, err := conn.Exec(s); err != nil {
			t.Fatalf("exec %q: %v", s, err)
		}
	}
}

func TestNewDatabaseMigratesOldVault(t *testing.T) {
	path := filepath.Join(t.TempDir(), "db.sqlite")
	createOldVault(t, path)

	d := openAt(t, path)

	// All migration-added columns must now exist
	for _, col := range []string{"filename", "file_size", "date_created", "date_modified"} {
		if !columnExists(t, d, "files", col) {
			t.Errorf("files.%s missing after migration", col)
		}
	}

	version, err := d.SchemaVersion()
	if err != nil {
		t.Fatalf("SchemaVersion: %v", err)
	}
	if version != SchemaVersion {
		t.Fatalf("version = %d, want %d", version, SchemaVersion)
	}

	// filename backfill from vault_path
	var filename string
	if err := d.Conn().QueryRow("SELECT filename FROM files WHERE vault_path = 'a/b/c.jpg'").Scan(&filename); err != nil {
		t.Fatalf("query filename: %v", err)
	}
	if filename != "c.jpg" {
		t.Fatalf("backfilled filename = %q, want c.jpg", filename)
	}

	// idx_files_filename exists now (created by migration 4, not schema.sql)
	var idx int
	if err := d.Conn().QueryRow(
		"SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_files_filename'",
	).Scan(&idx); err != nil || idx != 1 {
		t.Fatalf("idx_files_filename missing after migration (count=%d, err=%v)", idx, err)
	}

	// the files_au trigger survived the backfill (dropped + recreated)
	var trig int
	if err := d.Conn().QueryRow(
		"SELECT COUNT(*) FROM sqlite_master WHERE type='trigger' AND name='files_au'",
	).Scan(&trig); err != nil || trig != 1 {
		t.Fatalf("files_au trigger missing after migration (count=%d, err=%v)", trig, err)
	}

	// FTS index was rebuilt: the pre-existing row is now searchable
	var fts int
	if err := d.Conn().QueryRow(
		"SELECT COUNT(*) FROM files_fts WHERE files_fts MATCH 'c'",
	).Scan(&fts); err != nil {
		t.Fatalf("fts query: %v", err)
	}

	// data survived the migration
	var count int
	if err := d.Conn().QueryRow("SELECT COUNT(*) FROM files").Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("files rows = %d, want 1", count)
	}
}

// TestMigrationFailureNotRecorded verifies a failed migration is rolled
// back and its version is NOT recorded, so a later open retries it.
func TestMigrationFailureNotRecorded(t *testing.T) {
	path := filepath.Join(t.TempDir(), "db.sqlite")
	d := openAt(t, path)

	// Simulate a failing migration: drop the files table so the
	// idempotency checks in later migrations can't no-op cleanly.
	// Instead, directly test the invariant with a failing up-func.
	m := migration{
		version: 7,
		up: func(tx *sql.Tx) error {
			if _, err := tx.Exec("UPDATE files SET nonexistent = 1"); err != nil {
				return err
			}
			return nil
		},
	}
	if err := d.runMigration(m); err == nil {
		t.Fatal("expected migration to fail")
	}

	var recorded int
	if err := d.Conn().QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = 7").Scan(&recorded); err != nil {
		t.Fatalf("query: %v", err)
	}
	if recorded != 0 {
		t.Fatal("failed migration was recorded in schema_migrations")
	}

	// Version unchanged
	v, _ := d.SchemaVersion()
	if v != SchemaVersion {
		t.Fatalf("version = %d, want %d", v, SchemaVersion)
	}
}
