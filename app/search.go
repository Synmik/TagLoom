package app

import (
	"database/sql"
	"fmt"
	"os"
	"sort"
	"strings"
	"syscall"

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
func (a *App) GetFiles(filter db.FileFilter, sortOpts db.SortOpts, page, limit int) (*db.FilePage, error) {
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
			args = append(args, "%."+fmt_+"%")
		}
		conditions = append(conditions, "("+strings.Join(formatPlaceholders, " OR f.vault_path LIKE ")+")")
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
	sortField := sortOpts.Field
	sortOrder := sortOpts.Order
	if sortField == "" {
		sortField = "indexed_at"
	}
	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}

	// Determine if this is a filesystem-based sort (value not stored in DB)
	isFSSort := sortField == "date_created" || sortField == "file_size"

	// Map sort field to DB column (for in-DB sorting)
	sortColumn := map[string]string{
		"name":       "f.name",
		"rating":     "f.rating",
		"indexed_at": "f.indexed_at",
		"filename":   "f.vault_path",
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

	// Filesystem-based sort: fetch all matching IDs+paths, stat files, sort in-memory, then paginate
	if isFSSort {
		type idPath struct {
			id   int64
			path string
		}

		// Fetch all matching file IDs and paths
		allRows, err := a.db.Conn().Query(fmt.Sprintf("SELECT f.id, f.vault_path FROM files f %s", whereClause), countArgs...)
		if err != nil {
			return nil, fmt.Errorf("failed to query files for FS sort: %w", err)
		}
		var allFiles []idPath
		defer allRows.Close()
		for allRows.Next() {
			var ip idPath
			if err := allRows.Scan(&ip.id, &ip.path); err != nil {
				continue
			}
			allFiles = append(allFiles, ip)
		}

		// Stat each file and attach sort value
		type idPathStat struct {
			id      int64
			path    string
			size    int64
			created int64 // Unix nanoseconds
		}
		var stats []idPathStat
		for _, ip := range allFiles {
			info, err := os.Stat(ip.path)
			if err != nil {
				continue // skip deleted files
			}
			var created int64
			if sys, ok := info.Sys().(*syscall.Win32FileAttributeData); ok {
				ct := sys.CreationTime
				created = int64(ct.HighDateTime)<<32 | int64(ct.LowDateTime)
				created = (created-116444736000000000)*100
			} else {
				created = info.ModTime().UnixNano()
			}
			stats = append(stats, idPathStat{id: ip.id, path: ip.path, size: info.Size(), created: created})
		}

		// Sort in-memory
		switch sortField {
		case "file_size":
			if sortOrder == "asc" {
				sort.Slice(stats, func(i, j int) bool { return stats[i].size < stats[j].size })
			} else {
				sort.Slice(stats, func(i, j int) bool { return stats[i].size > stats[j].size })
			}
		case "date_created":
			if sortOrder == "asc" {
				sort.Slice(stats, func(i, j int) bool { return stats[i].created < stats[j].created })
			} else {
				sort.Slice(stats, func(i, j int) bool { return stats[i].created > stats[j].created })
			}
		}

		// Paginate: extract the IDs for this page
		offset := page * limit
		if offset >= len(stats) {
			return &db.FilePage{Files: nil, TotalCount: totalCount, Page: page, Limit: limit}, nil
		}
		end := offset + limit
		if end > len(stats) {
			end = len(stats)
		}
		pageStats := stats[offset:end]

		// Build ID list for query
		idPlaceholders := make([]string, len(pageStats))
		idArgs := make([]any, len(pageStats))
		for i, s := range pageStats {
			idPlaceholders[i] = "?"
			idArgs[i] = s.id
		}

		// Fetch full records in sorted order
		querySQL := fmt.Sprintf(`
			SELECT f.id, f.vault_path, f.thumbnail_path, f.name, f.notes, f.link,
			       f.rating, f.is_favorite, f.folder_path, f.indexed_at
			FROM files f WHERE f.id IN (%s)
		`, strings.Join(idPlaceholders, ","))
		rows, err := a.db.Conn().Query(querySQL, idArgs...)
		if err != nil {
			return nil, fmt.Errorf("failed to query files after FS sort: %w", err)
		}
		files, err := scanFiles(rows)
		if err != nil {
			return nil, err
		}

		// Re-order results to match the sorted ID sequence
		fileMap := make(map[int64]db.File, len(files))
		for _, f := range files {
			fileMap[f.ID] = f
		}
		orderedFiles := make([]db.File, 0, len(pageStats))
		for _, s := range pageStats {
			if f, ok := fileMap[s.id]; ok {
				orderedFiles = append(orderedFiles, f)
			}
		}

		return &db.FilePage{
			Files:      orderedFiles,
			TotalCount: totalCount,
			Page:       page,
			Limit:      limit,
		}, nil
	}

	// In-DB sort (name, rating, indexed_at, filename)
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

// GetOriginalFilePath returns the absolute path of the original file on disk.
func (a *App) GetOriginalFilePath(id int64) (string, error) {
	if a.db == nil {
		return "", fmt.Errorf("no vault open")
	}

	var vaultPath string
	err := a.db.Conn().QueryRow("SELECT vault_path FROM files WHERE id = ?", id).Scan(&vaultPath)
	if err != nil {
		return "", fmt.Errorf("file not found: %w", err)
	}
	return vaultPath, nil
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
// Handles nullable columns (thumbnail_path, name, notes, link) via *string.
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
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return files, nil
}
