// Package db provides the SQLite database layer for TagLoom vaults:
// connection setup, schema, and migrations.
package db

import (
	"database/sql"
	"embed"
	"fmt"
	"os"
	"strings"

	// Register the pure-Go SQLite driver (no CGO required).
	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaFS embed.FS

// SchemaVersion is the current schema version. Bump this and append a
// migration to the list below whenever the schema changes.
const SchemaVersion = 6

// Database wraps the standard database/sql connection for a vault.
type Database struct {
	conn *sql.DB
}

// NewDatabase creates a new SQLite database at the given path.
// It initializes the schema if the database is new and applies any
// pending migrations (tracked in the schema_migrations table).
func NewDatabase(dbPath string) (*Database, error) {
	// modernc.org/sqlite uses file: URI with query params for config
	// WAL mode allows concurrent readers; busy_timeout gives writers time to wait
	connStr := fmt.Sprintf("file:%s?_journal_mode=WAL&_cache_size=4096&_busy_timeout=30000", dbPath)

	conn, err := sql.Open("sqlite", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := conn.Ping(); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	schema, err := schemaFS.ReadFile("schema.sql")
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to read schema: %w", err)
	}

	// Idempotent — safe for new and existing databases. Only creates
	// missing tables/indexes/FTS; existing files rows keep their old
	// columns until the migrations below add them.
	if _, err := conn.Exec(string(schema)); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to execute schema: %w", err)
	}

	db := &Database{conn: conn}

	if err := db.runMigrations(); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return db, nil
}

// Close closes the database connection.
func (d *Database) Close() error {
	if d.conn != nil {
		return d.conn.Close()
	}
	return nil
}

// Conn returns the underlying *sql.DB connection.
func (d *Database) Conn() *sql.DB {
	return d.conn
}

// migration is a single schema change, applied in version order.
// All migrations must be idempotent: they run unconditionally on old
// vaults (which have no schema_migrations table yet) and may run on
// databases where the change was applied out-of-band.
type migration struct {
	version int
	up      func(tx *sql.Tx) error
}

// migrations is the ordered migration history. Append-only: never edit
// or remove an entry once shipped.
var migrations = []migration{
	{version: 1, up: migrateCaseInsensitiveTagIndexes},
	{version: 2, up: migrateAddDateModified},
	{version: 3, up: migrateCompositeIndexes},
	{version: 4, up: migrateAddFilenameAndDateCreated},
	{version: 5, up: migrateAddDateCreated},
	{version: 6, up: migrateAddFileSize},
}

// runMigrations applies all migrations newer than the recorded schema
// version. Old vaults have no schema_migrations table and start at 0.
func (d *Database) runMigrations() error {
	if _, err := d.conn.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			applied_at TEXT NOT NULL
		)`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	var current int
	if err := d.conn.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&current); err != nil {
		return fmt.Errorf("read schema version: %w", err)
	}

	for _, m := range migrations {
		if m.version <= current {
			continue
		}
		if err := d.runMigration(m); err != nil {
			return fmt.Errorf("apply migration %d: %w", m.version, err)
		}
	}
	return nil
}

// runMigration applies a single migration and records its version in
// the same transaction, so a failed migration is never marked applied.
func (d *Database) runMigration(m migration) error {
	tx, err := d.conn.Begin()
	if err != nil {
		return err
	}
	if err := m.up(tx); err != nil {
		_ = tx.Rollback() // transaction is already failed; nothing to do with the rollback error
		return err
	}
	if _, err := tx.Exec(
		"INSERT INTO schema_migrations (version, applied_at) VALUES (?, datetime('now'))",
		m.version,
	); err != nil {
		_ = tx.Rollback() // insert failed; nothing to do with the rollback error
		return err
	}
	return tx.Commit()
}

// migrateCaseInsensitiveTagIndexes ensures that existing databases have
// case-insensitive unique indexes on tags.name and tag_aliases.alias.
// New databases get these from schema.sql; this handles upgrades.
func migrateCaseInsensitiveTagIndexes(tx *sql.Tx) error {
	_, _ = tx.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_tags_name_nocase ON tags(LOWER(name))")
	_, err := tx.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_tag_aliases_alias_nocase ON tag_aliases(LOWER(alias))")
	return err
}

// migrateAddDateModified adds the date_modified column to the files table
// for existing databases that were created before the column existed.
func migrateAddDateModified(tx *sql.Tx) error {
	var exists int
	err := tx.QueryRow("SELECT COUNT(*) FROM pragma_table_info('files') WHERE name='date_modified'").Scan(&exists)
	if err != nil || exists > 0 {
		return nil // column already exists or can't check
	}
	_, err = tx.Exec("ALTER TABLE files ADD COLUMN date_modified TEXT NOT NULL DEFAULT ''")
	return err
}

// migrateCompositeIndexes adds composite indexes for common filter combinations.
// New databases get these from schema.sql; this handles upgrades.
// NOTE: idx_files_filename is NOT created here — it depends on the filename
// column, so it is created by migration 4 after that column exists.
func migrateCompositeIndexes(tx *sql.Tx) error {
	_, _ = tx.Exec("CREATE INDEX IF NOT EXISTS idx_files_folder_fav_rating ON files(folder_path, is_favorite, rating, id)")
	_, _ = tx.Exec("CREATE INDEX IF NOT EXISTS idx_files_fav_rating ON files(is_favorite, rating, id)")
	_, err := tx.Exec("CREATE INDEX IF NOT EXISTS idx_file_tags_file ON file_tags(file_id)")
	return err
}

// migrateAddFilenameAndDateCreated adds the filename column
// for existing databases. New databases get this from schema.sql.
// It also creates idx_files_filename — the index depends on the column, so
// it must only be created once the column is guaranteed to exist (schema.sql
// and migration 3 run before this on old vaults).
func migrateAddFilenameAndDateCreated(tx *sql.Tx) error {
	var exists int
	if err := tx.QueryRow("SELECT COUNT(*) FROM pragma_table_info('files') WHERE name='filename'").Scan(&exists); err != nil {
		return nil // can't check; retry on next open
	}
	if exists == 0 {
		// Add filename column
		if _, err := tx.Exec("ALTER TABLE files ADD COLUMN filename TEXT NOT NULL DEFAULT ''"); err != nil {
			return fmt.Errorf("failed to add filename column: %w", err)
		}
		// Backfill: extract basename from vault_path.
		// Done in Go because SQLite has no "last separator" function; the
		// old substr/replace formula could not compute it and silently
		// produced garbage (e.g. "g" for "a/b/c.jpg").
		if err := backfillFilenames(tx); err != nil {
			fmt.Fprintf(os.Stderr, "filename backfill failed, will populate on next scan: %v\n", err)
		}
		// Rebuild the FTS index: vaults that predate FTS have an empty index
		// for existing rows; for in-sync vaults this is a redundant no-op.
		if _, err := tx.Exec(`INSERT INTO files_fts(files_fts) VALUES('rebuild')`); err != nil {
			fmt.Fprintf(os.Stderr, "fts rebuild failed, search results may be stale until rescan: %v\n", err)
		}
	}
	_, err := tx.Exec("CREATE INDEX IF NOT EXISTS idx_files_filename ON files(filename)")
	return err
}

// ftsUpdateTriggerDDL mirrors the files_au trigger in schema.sql. The
// backfill UPDATE must not fire it: on vaults where the FTS index was
// created after the files rows existed, the trigger's 'delete' of the old
// (never-indexed) values corrupts the FTS index (SQLITE_CORRUPT), breaking
// the UPDATE and all later FTS writes. So the trigger is dropped before
// the backfill and recreated afterwards. Vaults whose FTS index is already
// in sync keep correct behaviour either way.
const ftsUpdateTriggerDDL = `CREATE TRIGGER IF NOT EXISTS files_au AFTER UPDATE ON files BEGIN
    INSERT INTO files_fts(files_fts, rowid, name, notes) VALUES('delete', old.id, old.name, old.notes);
    INSERT INTO files_fts(rowid, name, notes) VALUES (new.id, new.name, new.notes);
END`

// backfillFilenames sets filename to the basename of vault_path for rows
// where it is still empty. The backfill never changes name/notes, so the
// FTS index is left untouched (see ftsUpdateTriggerDDL).
func backfillFilenames(tx *sql.Tx) error {
	if _, err := tx.Exec("DROP TRIGGER IF EXISTS files_au"); err != nil {
		return fmt.Errorf("drop files_au: %w", err)
	}

	rows, err := tx.Query("SELECT id, vault_path FROM files WHERE filename = ''")
	if err != nil {
		return fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	upd, err := tx.Prepare("UPDATE files SET filename = ? WHERE id = ?")
	if err != nil {
		return fmt.Errorf("prepare: %w", err)
	}
	defer upd.Close()

	var id int64
	var vaultPath string
	for rows.Next() {
		if err := rows.Scan(&id, &vaultPath); err != nil {
			return fmt.Errorf("scan: %w", err)
		}
		if _, err := upd.Exec(baseName(vaultPath), id); err != nil {
			return fmt.Errorf("exec: %w", err)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("rows: %w", err)
	}
	// Recreate the FTS update trigger dropped above — the backfill ran while
	// it was absent. A failure here must abort the migration.
	if _, err := tx.Exec(ftsUpdateTriggerDDL); err != nil {
		return fmt.Errorf("recreate files_au: %w", err)
	}
	return nil
}

// baseName returns the last path segment, handling both / and \\\n// separators regardless of host OS (vault paths are stored with the
// separator of the OS that created them).
func baseName(p string) string {
	if i := strings.LastIndexAny(p, "\\/"); i >= 0 {
		return p[i+1:]
	}
	return p
}

// migrateAddDateCreated adds the date_created column for existing databases.
// New databases get this from schema.sql.
// IMPORTANT: does NOT backfill — date_created should come from real filesystem
// creation time (Windows Win32FileAttributeData), not from date_modified.
// Existing rows without a date_created will get it populated on next re-scan
// or on-demand via GetFileMetadata fallback.
func migrateAddDateCreated(tx *sql.Tx) error {
	var exists int
	err := tx.QueryRow("SELECT COUNT(*) FROM pragma_table_info('files') WHERE name='date_created'").Scan(&exists)
	if err != nil || exists > 0 {
		return nil // column already exists
	}
	_, err = tx.Exec("ALTER TABLE files ADD COLUMN date_created TEXT NOT NULL DEFAULT ''")
	if err != nil {
		return fmt.Errorf("failed to add date_created column: %w", err)
	}
	return nil
}

// migrateAddFileSize adds the file_size column for existing databases.
// New databases get this from schema.sql.
// IMPORTANT: does NOT backfill — file_size comes from the real filesystem at
// scan time; rows without a size (0) get it populated on the next rescan.
func migrateAddFileSize(tx *sql.Tx) error {
	var exists int
	err := tx.QueryRow("SELECT COUNT(*) FROM pragma_table_info('files') WHERE name='file_size'").Scan(&exists)
	if err != nil || exists > 0 {
		return nil // column already exists
	}
	_, err = tx.Exec("ALTER TABLE files ADD COLUMN file_size INTEGER NOT NULL DEFAULT 0")
	if err != nil {
		return fmt.Errorf("failed to add file_size column: %w", err)
	}
	return nil
}

// SchemaVersion returns the applied schema version of this database.
func (d *Database) SchemaVersion() (int, error) {
	var version int
	err := d.conn.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&version)
	return version, err
}

// SeedDefaultTags is a no-op — no default tags are seeded.
// Users create their own tags via the TagManagerModal.
func (d *Database) SeedDefaultTags() error {
	return nil
}
