package db

import (
	"embed"
	"database/sql"
	"fmt"

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

// SeedDefaultTags is a no-op — no default tags are seeded.
// Users create their own tags via the TagManagerModal.
func (d *Database) SeedDefaultTags() error {
	return nil
}
