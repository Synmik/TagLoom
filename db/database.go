package db

import (
	"embed"
	"database/sql"
	"fmt"
	"log"

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
	// ?_journal_mode=WAL&_cache_size=4096&_busy_timeout=5000
	connStr := fmt.Sprintf("file:%s?_journal_mode=WAL&_cache_size=4096&_busy_timeout=5000", dbPath)
	
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

	return &Database{conn: conn}, nil
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

// SeedDefaultTags inserts default meta tags and color tags if the tags table is empty.
func (d *Database) SeedDefaultTags() error {
	var count int
	err := d.conn.QueryRow("SELECT COUNT(*) FROM tags").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to count tags: %w", err)
	}
	if count > 0 {
		return nil // Already seeded
	}

	log.Println("Seeding default tags...")
	stmt, err := d.conn.Prepare(`
		INSERT INTO tags (name, color, is_category, sort_order, created_at)
		VALUES (?, ?, 0, ?, datetime('now'))
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare seed statement: %w", err)
	}
	defer stmt.Close()

	defaultTags := []struct {
		Name  string
		Color string
		Order int
	}{
		{"★★", "", 1},
		{"★★★", "", 2},
		{"★★★★", "", 3},
		{"★★★★★", "", 4},
		{"♥ Favorite", "#FF4444", 5},
		{"🟥 Red", "#FF0000", 10},
		{"🟦 Blue", "#0000FF", 11},
		{"🟩 Green", "#00FF00", 12},
		{"🟨 Yellow", "#FFFF00", 13},
		{"🟪 Purple", "#800080", 14},
		{"⬛ Black", "#000000", 15},
		{"⬜ White", "#FFFFFF", 16},
	}

	for _, t := range defaultTags {
		if _, err := stmt.Exec(t.Name, t.Color, t.Order); err != nil {
			return fmt.Errorf("failed to insert tag %q: %w", t.Name, err)
		}
	}
	return nil
}
