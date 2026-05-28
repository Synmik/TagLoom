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

	// Build WHERE clause dynamically
	var conditions []string
	var args []any
	argIdx := 1

	if filter.FolderPath != "" {
		conditions = append(conditions, fmt.Sprintf("f.folder_path = ?"))
		args = append(args, filter.FolderPath)
		argIdx++
	}
	if len(filter.TagIDs) > 0 {
		// Files must have ALL specified tags (AND logic)
		for _, tagID := range filter.TagIDs {
			conditions = append(conditions, fmt.Sprintf("f.id IN (SELECT file_id FROM file_tags WHERE tag_id = ?)"))
			args = append(args, tagID)
			argIdx++
		}
	}
	if len(filter.FileFormats) > 0 {
		formatPlaceholders := make([]string, len(filter.FileFormats))
		for i, fmt_ := range filter.FileFormats {
			formatPlaceholders[i] = "?"
			args = append(args, "%"+fmt_)
		}
		conditions = append(conditions, fmt.Sprintf("f.vault_path LIKE %s OR f.vault_path LIKE %s", 
			strings.Join(formatPlaceholders, " OR f.vault_path LIKE "),
			strings.Join(formatPlaceholders, " OR f.vault_path LIKE ")))
		// Simpler approach: just use the placeholders directly
		conditions[len(conditions)-1] = "(" + strings.Join(formatPlaceholders, " OR f.vault_path LIKE ") + ")"
	}
	if filter.MinRating > 0 {
		conditions = append(conditions, "f.rating >= ?")
		args = append(args, filter.MinRating)
		argIdx++
	}
	if filter.FavoritesOnly {
		conditions = append(conditions, "f.is_favorite = 1")
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Build sorting
	sortField := sort.Field
	sortOrder := sort.Order
	if sortField == "" {
		sortField = "indexed_at"
	}
	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}

	// Map sort field to DB column
	sortColumn := map[string]string{
		"name":       "f.name",
		"rating":     "f.rating",
		"indexed_at": "f.indexed_at",
		"filename":   "f.vault_path",
		"date_created": "f.vault_path", // fallback, actual date requires filesystem
		"file_size":  "f.vault_path",   // fallback, actual size requires filesystem
	}[sortField]
	if sortColumn == "" {
		sortColumn = "f.indexed_at"
	}

	// Count total matching files
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM files f %s", whereClause)
	var totalCount int
	countArgs := make([]any, len(args))
	copy(countArgs, args)
	row := a.db.Conn().QueryRow(countQuery, countArgs...)
	_ = row.Scan(&totalCount)

	// Fetch paginated results
	querySQL := fmt.Sprintf(`
		SELECT f.id, f.vault_path, f.thumbnail_path, f.name, f.notes, f.link,
		       f.rating, f.is_favorite, f.folder_path, f.indexed_at
		FROM files f %s
		ORDER BY %s %s
		LIMIT ? OFFSET ?
	`, whereClause, sortColumn, sortOrder)

	queryArgs := append(args, limit, page*limit)
	rows, err := a.db.Conn().Query(querySQL, queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query files: %w", err)
	}

	files, err := scanFiles(rows)
	if err != nil {
		return nil, err
	}

	return &db.FilePage{
		Files:      files,
		TotalCount: totalCount,
		Page:       page,
		Limit:      limit,
	}, nil
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
