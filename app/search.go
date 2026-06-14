package app

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"TagLoom/db"
	"TagLoom/utils"
)

// SearchFiles performs a full-text search across file names, user-set names, notes, and tags.
func (a *App) SearchFiles(query string, limit int) ([]db.File, error) {
	if a.db == nil {
		return nil, fmt.Errorf("no vault open")
	}

	if limit <= 0 {
		limit = 500
	}

	// FTS5 search on name and notes — split into words so each term is searched
	// (FTS5 uses space as a term separator, so "red sunset" matches both terms)
	ftsWords := strings.Fields(query)
	for i, w := range ftsWords {
		ftsWords[i] = sanitizeFTSQuery(w)
	}
	ftsQuery := strings.Join(ftsWords, " ")
	if ftsQuery == "" {
		ftsQuery = "*"
	}

	// Tag search: split query into words and match each against tag names and aliases.
	// "red sunset" → find files tagged with "red" OR "sunset" (or any partial match).
	tagWords := strings.Fields(query)
	var tagConditions []string
	for _, w := range tagWords {
		if len(w) < 2 {
			continue
		}
		// Escape LIKE special characters in the word
		escaped := strings.ReplaceAll(w, "%", "\\%")
		escaped = strings.ReplaceAll(escaped, "_", "\\_")
		pattern := "%" + escaped + "%"
		tagConditions = append(tagConditions,
			fmt.Sprintf("(LOWER(t.name) LIKE %q OR LOWER(a.alias) LIKE %q)", pattern, pattern),
		)
	}
	// Cap tag conditions to avoid excessively large queries
	if len(tagConditions) > 10 {
		tagConditions = tagConditions[:10]
	}

	// Build the full query with all match conditions
	if len(tagConditions) > 0 {
		tagClause := "f.id IN (SELECT ft.file_id FROM file_tags ft JOIN tags t ON ft.tag_id = t.id LEFT JOIN tag_aliases a ON a.tag_id = t.id WHERE " +
			strings.Join(tagConditions, " OR ") + ")"

		querySQL := fmt.Sprintf(`
			SELECT f.id, f.vault_path, f.thumbnail_path, f.name, f.notes, f.link,
			       f.rating, f.is_favorite, f.folder_path, f.filename, f.date_created, f.date_modified, f.indexed_at
			FROM files f
			WHERE f.id IN (
				SELECT rowid FROM files_fts WHERE files_fts MATCH %q
			)
			OR f.vault_path LIKE %q
			OR %s
			LIMIT ?
		`, ftsQuery, "%"+query+"%", tagClause)

		rows, err := a.db.Conn().Query(querySQL, limit)
		if err != nil {
			return nil, err
		}
		return scanFiles(rows)
	}

	// No valid tag words — fall back to FTS + filename only
	querySQL := fmt.Sprintf(`
		SELECT f.id, f.vault_path, f.thumbnail_path, f.name, f.notes, f.link,
		       f.rating, f.is_favorite, f.folder_path, f.filename, f.date_created, f.date_modified, f.indexed_at
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
		// Convert absolute folder path to relative for DB comparison.
		// Root folder (vault path) → ".", subfolder → relative path.
		folderFilter := filter.FolderPath
		if filepath.IsAbs(folderFilter) {
			folderFilter = a.toRelativePath(folderFilter)
		}
		args = append(args, folderFilter)
		argIdx++
	}
	if len(filter.TagGroups) > 0 {
		// Each group = OR (tag + descendants); between groups = AND
		for _, group := range filter.TagGroups {
			if len(group) == 0 {
				continue
			}
			if len(group) == 1 {
				// Single tag — simple equality
				conditions = append(conditions, "f.id IN (SELECT file_id FROM file_tags WHERE tag_id = ?)")
				args = append(args, group[0])
			} else {
				// Multiple tags in group — OR via IN
				placeholders := make([]string, len(group))
				for i, tagID := range group {
					placeholders[i] = "?"
					args = append(args, tagID)
				}
				conditions = append(conditions, fmt.Sprintf("f.id IN (SELECT file_id FROM file_tags WHERE tag_id IN (%s))", strings.Join(placeholders, ",")))
			}
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
	if filter.UntaggedOnly {
		conditions = append(conditions, "f.id NOT IN (SELECT file_id FROM file_tags)")
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
	isFSSort := sortField == "file_size"

	// Map sort field to DB column (for in-DB sorting)
	sortColumn := map[string]string{
		"name":          "f.name",
		"rating":        "f.rating",
		"indexed_at":    "f.indexed_at",
		"filename":      "f.filename",
		"date_created":  "f.date_created",
		"date_modified": "f.date_modified",
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
			absPath := a.resolvePath(ip.path)
			info, err := os.Stat(absPath)
			if err != nil {
				continue // skip deleted files
			}
			created := utils.GetCreationTimeNanos(absPath)
			stats = append(stats, idPathStat{id: ip.id, path: absPath, size: info.Size(), created: created})
		}

		// Sort in-memory
		switch sortField {
		case "file_size":
			if sortOrder == "asc" {
				sort.Slice(stats, func(i, j int) bool { return stats[i].size < stats[j].size })
			} else {
				sort.Slice(stats, func(i, j int) bool { return stats[i].size > stats[j].size })
			}
		case "date_modified":
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
			       f.rating, f.is_favorite, f.folder_path, f.filename, f.date_created, f.date_modified, f.indexed_at
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

		// Attach tags to each file
		attachTagsForFiles(a.db, files)

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

	// In-DB sort
	querySQL := fmt.Sprintf(`
		SELECT f.id, f.vault_path, f.thumbnail_path, f.name, f.notes, f.link,
		       f.rating, f.is_favorite, f.folder_path, f.filename, f.date_created, f.date_modified, f.indexed_at
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

	// Batch-fetch tags for all files in this page
	attachTagsForFiles(a.db, files)

	return &db.FilePage{
		Files:      files,
		TotalCount: totalCount,
		Page:       page,
		Limit:      limit,
	}, nil
}

// attachTagsForFiles does a single batch query to get all tags for the given file IDs
// and attaches them to each file's Tags slice.
func attachTagsForFiles(database *db.Database, files []db.File) {
	if len(files) == 0 {
		return
	}

	fileIDs := make([]int64, len(files))
	for i, f := range files {
		fileIDs[i] = f.ID
	}

	tagsMap, err := getTagsForFiles(database, fileIDs)
	if err != nil {
		return // non-fatal: files returned without tags
	}
	for i := range files {
		files[i].Tags = tagsMap[files[i].ID]
	}
}

// getTagsForFiles returns a map of fileID -> []Tag for the given file IDs.
func getTagsForFiles(database *db.Database, fileIDs []int64) (map[int64][]db.Tag, error) {
	tagsMap := make(map[int64][]db.Tag)

	placeholders := make([]string, len(fileIDs))
	args := make([]any, len(fileIDs))
	for i, id := range fileIDs {
		placeholders[i] = "?"
		args[i] = id
	}

	querySQL := fmt.Sprintf(`
		SELECT ft.file_id, t.id, t.name, t.color, t.parent_id, t.is_category, t.sort_order, t.created_at
		FROM file_tags ft
		JOIN tags t ON ft.tag_id = t.id
		WHERE ft.file_id IN (%s)
		ORDER BY ft.file_id, t.sort_order
	`, strings.Join(placeholders, ","))

	rows, err := database.Conn().Query(querySQL, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var fileID int64
		var t db.Tag
		var color sql.NullString
		var parentID sql.NullInt64

		if err := rows.Scan(&fileID, &t.ID, &t.Name, &color, &parentID,
			&t.IsCategory, &t.SortOrder, &t.CreatedAt); err != nil {
			continue
		}
		if color.Valid {
			t.Color = color.String
		}
		if parentID.Valid {
			val := parentID.Int64
			t.ParentID = &val
		}
		tagsMap[fileID] = append(tagsMap[fileID], t)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tagsMap, nil
}

// GetOriginalFilePath returns the absolute path of the original file on disk.
// The vault_path in DB is relative; this resolves it to an absolute path.
func (a *App) GetOriginalFilePath(id int64) (string, error) {
	if a.db == nil {
		return "", fmt.Errorf("no vault open")
	}

	var relPath string
	err := a.db.Conn().QueryRow("SELECT vault_path FROM files WHERE id = ?", id).Scan(&relPath)
	if err != nil {
		return "", fmt.Errorf("file not found: %w", err)
	}
	return a.resolvePath(relPath), nil
}

// GetFileByID returns a single file by its ID.
func (a *App) GetFileByID(id int64) (*db.File, error) {
	if a.db == nil {
		return nil, fmt.Errorf("no vault open")
	}

	row := a.db.Conn().QueryRow(`
		SELECT id, vault_path, thumbnail_path, name, notes, link,
		       rating, is_favorite, folder_path, filename, date_created, date_modified, indexed_at
		FROM files WHERE id = ?
	`, id)

	var file db.File
	if err := row.Scan(&file.ID, &file.VaultPath, &file.ThumbnailPath, &file.Name,
		&file.Notes, &file.Link, &file.Rating, &file.IsFavorite,
		&file.FolderPath, &file.Filename, &file.DateCreated, &file.DateModified, &file.IndexedAt); err != nil {
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
			&f.FolderPath, &f.Filename, &f.DateCreated, &f.DateModified, &f.IndexedAt); err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return files, nil
}
