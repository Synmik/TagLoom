package app

import (
	"database/sql"
	"fmt"
	"strings"

	"TagLoom/db"
)

// SearchFiles performs a full-text search across file names, user-set names, notes, and tags.
func (a *App) SearchFiles(query string, limit int) ([]db.File, error) {
	if a.db == nil {
		return nil, fmt.Errorf("no vault open")
	}

	if limit <= 0 {
		limit = 100
	}

	// FTS5 search on name and notes
	// Filename search is handled via vault_path LIKE match
	ftsQuery := sanitizeFTSQuery(query)

	querySQL := fmt.Sprintf(`
		SELECT f.id, f.vault_path, f.thumbnail_path, f.name, f.notes, f.link,
		       f.rating, f.is_favorite, f.folder_path, f.indexed_at
		FROM files f
		WHERE f.id IN (
			SELECT rowid FROM files_fts WHERE files_fts MATCH %q
		)
		OR f.vault_path LIKE %q
		LIMIT ?
	`, ftsQuery, "%"+query+"%")

	rows, err := a.db.Conn().Query(querySQL, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanFiles(rows)
}

// GetFiles returns a paginated list of files with optional filters and sorting.
func (a *App) GetFiles(filter db.FileFilter, sort db.SortOpts, page, limit int) (*db.FilePage, error) {
	if a.db == nil {
		return nil, fmt.Errorf("no vault open")
	}

	if limit <= 0 {
		limit = 50
	}
	if page < 0 {
		page = 0
	}
	// TODO: Build dynamic query with filters (folder, tags, format, rating, favorites)
	// TODO: Apply sorting (DB fields vs filesystem fields)
	// TODO: Return paginated result
	return nil, fmt.Errorf("not implemented")
}

// GetFileByID returns a single file by its ID.
func (a *App) GetFileByID(id int64) (*db.File, error) {
	if a.db == nil {
		return nil, fmt.Errorf("no vault open")
	}

	row := a.db.Conn().QueryRow(`
		SELECT id, vault_path, thumbnail_path, name, notes, link,
		       rating, is_favorite, folder_path, indexed_at
		FROM files WHERE id = ?
	`, id)

	var file db.File
	if err := row.Scan(&file.ID, &file.VaultPath, &file.ThumbnailPath, &file.Name,
		&file.Notes, &file.Link, &file.Rating, &file.IsFavorite,
		&file.FolderPath, &file.IndexedAt); err != nil {
		return nil, err
	}
	return &file, nil
}

// UpdateFile updates user-editable fields of a file.
func (a *App) UpdateFile(update *db.FileUpdate) error {
	if a.db == nil {
		return fmt.Errorf("no vault open")
	}

	_, err := a.db.Conn().Exec(`
		UPDATE files SET
			name = COALESCE(?, name),
			notes = COALESCE(?, notes),
			link = COALESCE(?, link),
			rating = ?,
			is_favorite = ?
		WHERE id = ?
	`, update.Name, update.Notes, update.Link, update.Rating, update.IsFavorite, update.ID)
	return err
}

// sanitizeFTSQuery escapes special FTS5 characters.
func sanitizeFTSQuery(query string) string {
	// Escape: * ? " [ ] ( ) { } ~ : \ & | -
	result := strings.ReplaceAll(query, "\\", "\\\\")
	result = strings.ReplaceAll(result, "\"", "\\\"")
	result = strings.ReplaceAll(result, "*", "\\*")
	result = strings.ReplaceAll(result, "?", "\\?")
	return result
}

// scanFiles scans database rows into a slice of File structs.
func scanFiles(rows *sql.Rows) ([]db.File, error) {
	var files []db.File
	defer rows.Close()
	for rows.Next() {
		var f db.File
		if err := rows.Scan(&f.ID, &f.VaultPath, &f.ThumbnailPath, &f.Name,
			&f.Notes, &f.Link, &f.Rating, &f.IsFavorite,
			&f.FolderPath, &f.IndexedAt); err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	return files, nil
}
