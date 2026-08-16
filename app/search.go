package app

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"

	"TagLoom/db"
)

// SearchFiles performs a full-text search across file names, user-set names, notes, and tags.
func (a *App) SearchFiles(query string, limit int) ([]db.File, error) {
	v := a.vault()
	if v.db == nil {
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
		// Empty phrase — matches no rows, but keeps the query valid.
		// (A bare `*` match-all is not supported by all FTS5 builds.)
		ftsQuery = `""`
	}

	// Filename search via LIKE — wildcards in the input are escaped so the
	// pattern matches its input literally.
	pathPattern := "%" + escapeLike(query) + "%"

	// Tag search: split query into words and match each against tag names and aliases.
	// "red sunset" → find files tagged with "red" OR "sunset" (or any partial match).
	tagWords := strings.Fields(query)
	var tagConditions []string
	var tagArgs []any
	for _, w := range tagWords {
		if len(w) < 2 {
			continue
		}
		pattern := "%" + escapeLike(w) + "%"
		tagConditions = append(tagConditions, "(LOWER(t.name) LIKE ? ESCAPE '\\' OR LOWER(a.alias) LIKE ? ESCAPE '\\')")
		tagArgs = append(tagArgs, pattern, pattern)
	}
	// Cap tag conditions to avoid excessively large queries
	if len(tagConditions) > 10 {
		tagConditions = tagConditions[:10]
		tagArgs = tagArgs[:20]
	}

	// Build the full query with all match conditions. All user input goes through
	// bound parameters — ftsQuery is a safe FTS5 phrase (sanitizeFTSQuery) and the
	// LIKE patterns are wildcard-escaped (escapeLike).
	whereParts := []string{
		"f.id IN (SELECT rowid FROM files_fts WHERE files_fts MATCH ?)",
		"f.vault_path LIKE ? ESCAPE '\\'",
	}
	args := []any{ftsQuery, pathPattern}

	if len(tagConditions) > 0 {
		whereParts = append(whereParts, "f.id IN (SELECT ft.file_id FROM file_tags ft JOIN tags t ON ft.tag_id = t.id LEFT JOIN tag_aliases a ON a.tag_id = t.id WHERE " +
			strings.Join(tagConditions, " OR ") + ")")
		args = append(args, tagArgs...)
	}
	args = append(args, limit)

	querySQL := "\n"
	querySQL += `SELECT f.id, f.vault_path, f.thumbnail_path, f.name, f.notes, f.link,
	       f.rating, f.is_favorite, f.folder_path, f.filename, f.file_size,
	       f.date_created, f.date_modified, f.indexed_at
	FROM files f
	WHERE ` + strings.Join(whereParts, " OR ") + `
	LIMIT ?`

	rows, err := v.db.Conn().Query(querySQL, args...)
	if err != nil {
		return nil, err
	}

	return scanFiles(rows)
}

// GetFiles returns a paginated list of files with optional filters and sorting.
func (a *App) GetFiles(filter db.FileFilter, sortOpts db.SortOpts, page, limit int) (*db.FilePage, error) {
	v := a.vault()
	if v.db == nil {
		return nil, fmt.Errorf("no vault open")
	}

	if limit <= 0 {
		limit = 50
	}
	if page < 0 {
		page = 0
	}

	whereClause, args := buildFileFilterClause(v, filter)

	// Build sorting
	sortField := sortOpts.Field
	sortOrder := sortOpts.Order
	if sortField == "" {
		sortField = "indexed_at"
	}
	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}

	// Map sort field to DB column — all sort values are stored in the DB
	// (file_size is populated at scan/import time, so no filesystem stats
	// are needed per page request).
	sortColumn := map[string]string{
		"name":          "f.name",
		"rating":        "f.rating",
		"indexed_at":    "f.indexed_at",
		"filename":      "f.filename",
		"file_size":     "f.file_size",
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
	row := v.db.Conn().QueryRow(countQuery, countArgs...)
	_ = row.Scan(&totalCount)

	// In-DB sort
	querySQL := fmt.Sprintf(`
		SELECT f.id, f.vault_path, f.thumbnail_path, f.name, f.notes, f.link,
		       f.rating, f.is_favorite, f.folder_path, f.filename, f.file_size,
		       f.date_created, f.date_modified, f.indexed_at
		FROM files f %s
		ORDER BY %s %s
		LIMIT ? OFFSET ?
	`, whereClause, sortColumn, sortOrder)

	queryArgs := append(args, limit, page*limit)
	rows, err := v.db.Conn().Query(querySQL, queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query files: %w", err)
	}

	files, err := scanFiles(rows)
	if err != nil {
		return nil, err
	}

	// Batch-fetch tags for all files in this page
	attachTagsForFiles(v.db, files)

	return &db.FilePage{
		Files:      files,
		TotalCount: totalCount,
		Page:       page,
		Limit:      limit,
	}, nil
}

// buildFileFilterClause builds the WHERE clause (with bound arguments) for a
// files query from the given filter. Shared by GetFiles (rows + count) and
// GetFileIDs (ids only), so both always agree on what "matching" means.
func buildFileFilterClause(v vault, filter db.FileFilter) (string, []any) {
	var conditions []string
	var args []any

	if filter.FolderPath != "" {
		conditions = append(conditions, fmt.Sprintf("f.folder_path = ?"))
		// Convert absolute folder path to relative for DB comparison.
		// Root folder (vault path) → ".", subfolder → relative path.
		folderFilter := filter.FolderPath
		if filepath.IsAbs(folderFilter) {
			folderFilter = v.toRelativePath(folderFilter)
		}
		args = append(args, folderFilter)
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
		// LOWER() on both sides — SQLite LIKE is case-sensitive for ASCII,
		// and stored paths can carry uppercase extensions.
		// One full "LOWER(f.vault_path) LIKE ?" clause per format — the
		// join must never supply the LIKE keyword, or a single-format
		// filter degenerates into WHERE (?).
		formatParts := make([]string, len(filter.FileFormats))
		for i, fmt_ := range filter.FileFormats {
			formatParts[i] = "LOWER(f.vault_path) LIKE ?"
			args = append(args, "%."+strings.ToLower(fmt_)+"%")
		}
		conditions = append(conditions, "("+strings.Join(formatParts, " OR ")+")")
	}
	if filter.MinRating > 0 {
		conditions = append(conditions, "f.rating >= ?")
		args = append(args, filter.MinRating)
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
	return whereClause, args
}

// GetFileIDs returns only the IDs of all files matching the filter — no
// pagination, sorting, COUNT(*), or tag attachment. Batch operations that
// target an entire folder need exactly this and nothing else, so a 500K-file
// folder costs one query returning bare ints instead of ~1000 pages of full
// file objects over the JSON-RPC bridge.
func (a *App) GetFileIDs(filter db.FileFilter) ([]int64, error) {
	v := a.vault()
	if v.db == nil {
		return nil, fmt.Errorf("no vault open")
	}

	whereClause, args := buildFileFilterClause(v, filter)
	querySQL := fmt.Sprintf("SELECT f.id FROM files f %s", whereClause)

	rows, err := v.db.Conn().Query(querySQL, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query file IDs: %w", err)
	}
	defer rows.Close()

	ids := []int64{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return ids, nil
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
	v := a.vault()
	if v.db == nil {
		return "", fmt.Errorf("no vault open")
	}

	var relPath string
	err := v.db.Conn().QueryRow("SELECT vault_path FROM files WHERE id = ?", id).Scan(&relPath)
	if err != nil {
		return "", fmt.Errorf("file not found: %w", err)
	}
	return v.resolvePath(relPath), nil
}

// GetFileByID returns a single file by its ID.
func (a *App) GetFileByID(id int64) (*db.File, error) {
	v := a.vault()
	if v.db == nil {
		return nil, fmt.Errorf("no vault open")
	}

	row := v.db.Conn().QueryRow(`
		SELECT id, vault_path, thumbnail_path, name, notes, link,
		       rating, is_favorite, folder_path, filename, file_size,
		       date_created, date_modified, indexed_at
		FROM files WHERE id = ?
	`, id)

	var file db.File
	if err := row.Scan(&file.ID, &file.VaultPath, &file.ThumbnailPath, &file.Name,
		&file.Notes, &file.Link, &file.Rating, &file.IsFavorite,
		&file.FolderPath, &file.Filename, &file.FileSize,
		&file.DateCreated, &file.DateModified, &file.IndexedAt); err != nil {
		return nil, err
	}
	return &file, nil
}

// UpdateFile updates user-editable fields of a file.
func (a *App) UpdateFile(update *db.FileUpdate) error {
	v := a.vault()
	if v.db == nil {
		return fmt.Errorf("no vault open")
	}

	_, err := v.db.Conn().Exec(`
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

// sanitizeFTSQuery turns a raw search word into a safe FTS5 phrase: the word is
// wrapped in double quotes, which makes every character literal text — only an
// embedded double quote needs escaping, per FTS5 query syntax, as "". The result
// is always safe to bind as a parameter to a MATCH expression.
func sanitizeFTSQuery(word string) string {
	return `"` + strings.ReplaceAll(word, `"`, `""`) + `"`
}

// escapeLike escapes the LIKE wildcards (%, _) and the escape character itself
// so the pattern matches its input literally. Always use the result together
// with the SQL clause `ESCAPE '\'`.
func escapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, "%", `\%`)
	s = strings.ReplaceAll(s, "_", `\_`)
	return s
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
			&f.FolderPath, &f.Filename, &f.FileSize,
			&f.DateCreated, &f.DateModified, &f.IndexedAt); err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return files, nil
}
