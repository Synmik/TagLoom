package db

import (
	"database/sql"
	"embed"
	"fmt"
	"os"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaFS embed.FS

// Database wraps the standard database/sql connection for a vault.
type Database struct {
	conn *sql.DB
}

// NewDatabase creates a new SQLite database at the given path.
// It initializes the schema if the database is new.
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
		conn.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	schema, err := schemaFS.ReadFile("schema.sql")
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to read schema: %w", err)
	}

	_, err = conn.Exec(string(schema))
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to execute schema: %w", err)
	}

	db := &Database{conn: conn}

	// Run migrations for existing databases (no-op for new ones)
	if err := db.MigrateCaseInsensitiveTagIndexes(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}
	if err := db.MigrateAddDateModified(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}
	if err := db.MigrateCompositeIndexes(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}
	if err := db.MigrateAddFilenameAndDateCreated(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}
	if err := db.MigrateAddDateCreated(); err != nil {
		conn.Close()
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

// MigrateCaseInsensitiveTagIndexes ensures that existing databases have
// case-insensitive unique indexes on tags.name and tag_aliases.alias.
// New databases get these from schema.sql; this handles upgrades.
func (d *Database) MigrateCaseInsensitiveTagIndexes() error {
	_, _ = d.conn.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_tags_name_nocase ON tags(LOWER(name))")
	_, err := d.conn.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_tag_aliases_alias_nocase ON tag_aliases(LOWER(alias))")
	return err
}

// MigrateAddDateModified adds the date_modified column to the files table
// for existing databases that were created before the column existed.
func (d *Database) MigrateAddDateModified() error {
	var exists int
	err := d.conn.QueryRow("SELECT COUNT(*) FROM pragma_table_info('files') WHERE name='date_modified'").Scan(&exists)
	if err != nil || exists > 0 {
		return nil // column already exists or can't check
	}
	_, err = d.conn.Exec("ALTER TABLE files ADD COLUMN date_modified TEXT NOT NULL DEFAULT ''")
	return err
}

// MigrateCompositeIndexes adds composite indexes for common filter combinations.
// New databases get these from schema.sql; this handles upgrades.
func (d *Database) MigrateCompositeIndexes() error {
	_, _ = d.conn.Exec("CREATE INDEX IF NOT EXISTS idx_files_folder_fav_rating ON files(folder_path, is_favorite, rating, id)")
	_, _ = d.conn.Exec("CREATE INDEX IF NOT EXISTS idx_files_fav_rating ON files(is_favorite, rating, id)")
	_, err := d.conn.Exec("CREATE INDEX IF NOT EXISTS idx_file_tags_file ON file_tags(file_id)")
	_, _ = d.conn.Exec("CREATE INDEX IF NOT EXISTS idx_files_filename ON files(filename)")
	return err
}

// MigrateAddFilenameAndDateCreated adds filename column
// for existing databases. New databases get this from schema.sql.
func (d *Database) MigrateAddFilenameAndDateCreated() error {
	var exists int
	err := d.conn.QueryRow("SELECT COUNT(*) FROM pragma_table_info('files') WHERE name='filename'").Scan(&exists)
	if err != nil || exists > 0 {
		return nil // column already exists or can't check
	}
	// Add filename column
	_, err = d.conn.Exec("ALTER TABLE files ADD COLUMN filename TEXT NOT NULL DEFAULT ''")
	if err != nil {
		return fmt.Errorf("failed to add filename column: %w", err)
	}
	// Backfill: extract basename from vault_path
	_, err = d.conn.Exec("UPDATE files SET filename = substr(vault_path, length(replace(vault_path, '/', '')) + 2)")
	if err != nil {
		fmt.Fprintf(os.Stderr, "filename backfill failed, will populate on next scan: %v\n", err)
	}
	return nil
}

// MigrateAddDateCreated adds the date_created column for existing databases.
// New databases get this from schema.sql.
// IMPORTANT: does NOT backfill — date_created should come from real filesystem
// creation time (Windows Win32FileAttributeData), not from date_modified.
// Existing rows without a date_created will get it populated on next re-scan
// or on-demand via GetFileMetadata fallback.
func (d *Database) MigrateAddDateCreated() error {
	var exists int
	err := d.conn.QueryRow("SELECT COUNT(*) FROM pragma_table_info('files') WHERE name='date_created'").Scan(&exists)
	if err != nil || exists > 0 {
		return nil // column already exists
	}
	_, err = d.conn.Exec("ALTER TABLE files ADD COLUMN date_created TEXT NOT NULL DEFAULT ''")
	if err != nil {
		return fmt.Errorf("failed to add date_created column: %w", err)
	}
	return nil
}

// SeedDefaultTags is a no-op — no default tags are seeded.
// Users create their own tags via the TagManagerModal.
func (d *Database) SeedDefaultTags() error {
	return nil
}
