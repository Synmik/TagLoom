package app

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"TagLoom/db"
	"TagLoom/utils"
)

// scanEntry holds a file path and its os.FileInfo collected during a single
// filepath.WalkDir pass. Keeping the info in memory avoids a second walk of
// the directory tree and an extra os.Stat per file.
type scanEntry struct {
	path string
	info os.FileInfo
}

// ScanVault performs a full re-scan of the vault directory.
// It discovers all supported media files and indexes them in a single
// directory walk — entries (path + FileInfo) are collected in-memory,
// then batch-inserted in a transaction.
// Emits "scan:progress" events with {current, total} payload.
// Returns the total number of indexed files.
func (a *App) ScanVault() (int, error) {
	v := a.vault()
	if v.db == nil {
		return 0, fmt.Errorf("no vault open")
	}
	if v.path == "" {
		return 0, fmt.Errorf("no vault path set")
	}

	// Load excluded folders into a set for O(1) lookup
	excludedFolders, err := a.GetExcludedFolders()
	if err != nil {
		return 0, fmt.Errorf("failed to load excluded folders: %w", err)
	}
	excludedSet := make(map[string]bool, len(excludedFolders))
	for _, fp := range excludedFolders {
		excludedSet[strings.ToLower(filepath.Clean(fp))] = true
	}

	// Single-pass: collect all supported files (path + info) in memory.
	// This avoids a second directory walk and uses entry.Info() instead
	// of a separate os.Stat() call.
	var entries []scanEntry
	err = filepath.WalkDir(v.path, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if isExcluded(path, v.path, excludedSet) {
				return filepath.SkipDir
			}
			return nil
		}
		if !utils.IsSupported(path) {
			return nil
		}

		info, infoErr := d.Info()
		if infoErr != nil {
			return nil // Skip files we can't stat
		}
		entries = append(entries, scanEntry{path: path, info: info})
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("failed to walk directory: %w", err)
	}

	// Batch-insert collected entries in a single transaction
	tx, err := v.db.Conn().Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}

	stmt, err := tx.Prepare(`
		INSERT INTO files (vault_path, folder_path, filename, file_size, date_created, date_modified, indexed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(vault_path) DO UPDATE SET
			folder_path = excluded.folder_path,
			filename = excluded.filename,
			file_size = excluded.file_size,
			date_created = excluded.date_created,
			date_modified = excluded.date_modified,
			indexed_at = excluded.indexed_at
	`)
	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	totalCount := len(entries)
	now := time.Now().Format(time.RFC3339)

	for i, e := range entries {
		ft := utils.GetFileTimes(e.info)
		createdAtStr := ""
		if !ft.CreatedAt.IsZero() {
			createdAtStr = ft.CreatedAt.Format(time.RFC3339)
		}
		_, execErr := stmt.Exec(
			v.toRelativePath(e.path),
			v.toRelativePath(filepath.Dir(e.path)),
			filepath.Base(e.path),
			e.info.Size(),
			createdAtStr,
			ft.ModifiedAt.Format(time.RFC3339),
			now,
		)
		if execErr != nil {
			tx.Rollback()
			return 0, fmt.Errorf("failed to insert %s: %w", e.path, execErr)
		}

		// Emit progress every 100 files
		if (i+1)%100 == 0 || i+1 == totalCount {
			a.emitEvent("scan:progress", map[string]int{
				"current": i + 1,
				"total":   totalCount,
			})
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Refresh the cached file count (upserts don't tell us how many were
	// actually new vs. updated, and rows for deleted files may linger).
	a.refreshFileCount(v)

	// Emit completion
	a.emitEvent("scan:progress", map[string]int{
		"current": totalCount,
		"total":   totalCount,
	})
	a.emitEvent("scan:complete", totalCount)

	return totalCount, nil
}

// isExcluded checks if a directory path is in the excluded set.
// Always skips ".tagloom" (internal metadata directory) regardless of user config.
// The excluded paths are relative to vault root; they are resolved to absolute
// for comparison with the absolute paths from filepath.WalkDir.
func isExcluded(path string, vaultPath string, excluded map[string]bool) bool {
	clean := strings.ToLower(filepath.Clean(path))
	// Always exclude .tagloom (and anything inside it) — it contains
	// thumbnails, DB, and config. Match on any path component so paths
	// nested under .tagloom are excluded too.
	for _, part := range strings.Split(clean, string(filepath.Separator)) {
		if part == ".tagloom" {
			return true
		}
	}
	for excl := range excluded {
		// Resolve relative excluded path to absolute for comparison
		exclAbs := excl
		if !filepath.IsAbs(excl) {
			exclAbs = filepath.Join(vaultPath, excl)
		}
		exclAbs = strings.ToLower(filepath.Clean(exclAbs))
		if clean == exclAbs || strings.HasPrefix(clean, exclAbs+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

// RescanVault performs a diff scan, detecting added and removed files.
// Uses a single directory walk (collecting path + info) to avoid redundant I/O.
// Returns the number of added files. Removed count is sent via rescan:complete event.
func (a *App) RescanVault() (int, error) {
	v := a.vault()
	if v.db == nil {
		return 0, fmt.Errorf("no vault open")
	}
	if v.path == "" {
		return 0, fmt.Errorf("no vault path set")
	}

	// Load excluded folders
	excludedFolders, err := a.GetExcludedFolders()
	if err != nil {
		return 0, fmt.Errorf("failed to load excluded folders: %w", err)
	}
	excludedSet := make(map[string]bool, len(excludedFolders))
	for _, fp := range excludedFolders {
		excludedSet[strings.ToLower(filepath.Clean(fp))] = true
	}

	// Step 1: Single-pass — collect all filesystem files (path + info).
	// Store relative paths for comparison with DB (vault_path is relative).
	var fsEntries []scanEntry
	fsRelPaths := make(map[string]int) // relative path → index in fsEntries
	err = filepath.WalkDir(v.path, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if isExcluded(path, v.path, excludedSet) {
				return filepath.SkipDir
			}
			return nil
		}
		if !utils.IsSupported(path) {
			return nil
		}

		info, infoErr := d.Info()
		if infoErr != nil {
			return nil
		}
		idx := len(fsEntries)
		fsEntries = append(fsEntries, scanEntry{path: path, info: info})
		fsRelPaths[v.toRelativePath(path)] = idx
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("failed to walk directory: %w", err)
	}

	// Step 2: Collect all files from the DB
	rows, err := v.db.Conn().Query("SELECT vault_path FROM files")
	if err != nil {
		return 0, fmt.Errorf("failed to query DB files: %w", err)
	}
	defer rows.Close()

	dbFiles := make(map[string]bool)
	for rows.Next() {
		var vp string
		if err := rows.Scan(&vp); err != nil {
			return 0, err
		}
		dbFiles[vp] = true
	}

	// Step 3: Compute diff — compare relative paths on both sides.
	var added []scanEntry
	for _, e := range fsEntries {
		relPath := v.toRelativePath(e.path)
		if !dbFiles[relPath] {
			added = append(added, e)
		}
	}

	var removed []string
	for p := range dbFiles {
		if _, ok := fsRelPaths[p]; !ok {
			removed = append(removed, p)
		}
	}

	// Emit diff summary
	a.emitEvent("rescan:diff", map[string]int{
		"added":   len(added),
		"removed": len(removed),
		"total":   len(fsEntries),
	})

	// Step 4: Insert new files and delete removed in a transaction
	tx, err := v.db.Conn().Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Insert new files — use info collected during walk (no os.Stat needed)
	if len(added) > 0 {
		stmt, err := tx.Prepare(`
			INSERT INTO files (vault_path, folder_path, filename, file_size, date_created, date_modified, indexed_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`)
		if err != nil {
			tx.Rollback()
			return 0, fmt.Errorf("failed to prepare insert: %w", err)
		}

		now := time.Now().Format(time.RFC3339)
		for i, e := range added {
			ft := utils.GetFileTimes(e.info)
			createdAtStr := ""
			if !ft.CreatedAt.IsZero() {
				createdAtStr = ft.CreatedAt.Format(time.RFC3339)
			}
			_, err := stmt.Exec(
				v.toRelativePath(e.path),
				v.toRelativePath(filepath.Dir(e.path)),
				filepath.Base(e.path),
				e.info.Size(),
				createdAtStr,
				ft.ModifiedAt.Format(time.RFC3339),
				now,
			)
			if err != nil {
				stmt.Close()
				tx.Rollback()
				return 0, fmt.Errorf("failed to insert %s: %w", e.path, err)
			}

			step := (i + 1) * 100 / len(added)
			if step%25 == 0 || i+1 == len(added) {
				a.emitEvent("rescan:progress", map[string]interface{}{
					"phase":   "adding",
					"current": i + 1,
					"total":   len(added),
				})
			}
		}
		stmt.Close()
	}

	// Delete removed files (and their tag associations).
	// SQLite FK cascade is not enabled, so file_tags rows must be removed
	// explicitly. `removed` holds vault paths (not file IDs), so match
	// file_tags via a subquery on vault_path.
	if len(removed) > 0 {
		delTagStmt, err := tx.Prepare(`
			DELETE FROM file_tags
			WHERE file_id IN (SELECT id FROM files WHERE vault_path = ?)
		`)
		if err != nil {
			tx.Rollback()
			return 0, fmt.Errorf("failed to prepare file_tags delete: %w", err)
		}

		delFileStmt, err := tx.Prepare("DELETE FROM files WHERE vault_path = ?")
		if err != nil {
			delTagStmt.Close()
			tx.Rollback()
			return 0, fmt.Errorf("failed to prepare delete: %w", err)
		}

		// Collect thumbnail paths before the rows are gone so the WebPs
		// can be removed from disk after commit (no orphan thumbnails).
		var removedThumbs []string

		for i, p := range removed {
			var thumbRel *string
			if err := tx.QueryRow("SELECT thumbnail_path FROM files WHERE vault_path = ?", p).Scan(&thumbRel); err == nil && thumbRel != nil && *thumbRel != "" {
				removedThumbs = append(removedThumbs, v.resolvePath(*thumbRel))
			}
			if _, err := delTagStmt.Exec(p); err != nil {
				delTagStmt.Close()
				delFileStmt.Close()
				tx.Rollback()
				return 0, fmt.Errorf("failed to delete file_tags for %s: %w", p, err)
			}
			if _, err := delFileStmt.Exec(p); err != nil {
				delTagStmt.Close()
				delFileStmt.Close()
				tx.Rollback()
				return 0, fmt.Errorf("failed to delete %s: %w", p, err)
			}
			step := (i + 1) * 100 / len(removed)
			if step%25 == 0 || i+1 == len(removed) {
				a.emitEvent("rescan:progress", map[string]interface{}{
					"phase":   "removing",
					"current": i + 1,
					"total":   len(removed),
				})
			}
		}
		delTagStmt.Close()
		delFileStmt.Close()

		// Remove thumbnails of the files that are gone from disk.
		for _, thumbAbs := range removedThumbs {
			deleteThumbnailFile(thumbAbs)
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit: %w", err)
	}

	// Update the cached file count from the exact diff.
	a.adjustFileCount(len(added) - len(removed))

	// Emit completion
	a.emitEvent("rescan:complete", map[string]int{
		"added":   len(added),
		"removed": len(removed),
	})

	return len(added), nil
}

// GetFolderTree returns the recursive folder tree for the vault.
func (a *App) GetFolderTree(path string) *db.FolderNode {
	v := a.vault()
	if v.db == nil {
		return &db.FolderNode{
			Path:      path,
			Name:      filepath.Base(path),
			FileCount: 0,
			Children:  []db.FolderNode{},
		}
	}

	// Query all unique folder paths with file counts
	rows, err := v.db.Conn().Query(`
		SELECT folder_path, COUNT(*) as cnt
		FROM files
		GROUP BY folder_path
		ORDER BY folder_path
	`)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to query folder tree: %v\n", err)
		return &db.FolderNode{
			Path:      path,
			Name:      filepath.Base(path),
			FileCount: 0,
			Children:  []db.FolderNode{},
		}
	}
	defer rows.Close()

	// Build a map of folder_path → file count.
	// Normalize "." (files directly in vault root) to the vault root path.
	folderCounts := make(map[string]int)
	for rows.Next() {
		var fPath string
		var count int
		if err := rows.Scan(&fPath, &count); err != nil {
			continue
		}
		if fPath == "." {
			fPath = path // vault root
		}
		folderCounts[fPath] += count
	}

	// Build a children map: parentPath → []childPath.
	// Also ensure intermediate folders (no direct files but contain subfolders) are included.
	childrenOf := make(map[string][]string)
	for fPath := range folderCounts {
		// Skip the vault root itself — it's the top of the tree, not a child of anything
		if fPath == path {
			continue
		}
		parentPath := resolveParent(fPath, path)
		childrenOf[parentPath] = append(childrenOf[parentPath], fPath)
	}

	// Discover intermediate folders: parents that exist as directory nodes but have
	// no files directly in them. Loop until no new folders are found.
	for {
		var newFolders []string
		for parent := range childrenOf {
			if _, exists := folderCounts[parent]; exists || parent == path {
				continue
			}
			// This parent folder has no files — it's an intermediate directory
			folderCounts[parent] = 0
			newFolders = append(newFolders, parent)
		}
		if len(newFolders) == 0 {
			break
		}
		for _, fPath := range newFolders {
			parentPath := resolveParent(fPath, path)
			childrenOf[parentPath] = append(childrenOf[parentPath], fPath)
		}
	}

	// Build the tree recursively from the children map
	return buildFolderTree(folderCounts, childrenOf, path, make(map[string]bool))
}

// buildFolderTree recursively constructs the folder tree.
// Uses a children-map + recursive approach to avoid value-copy bugs with Go slices
// when building nested trees from value-type []FolderNode.
// The `seen` map guards against cycles (e.g. a folder listed as its own child).
func buildFolderTree(counts map[string]int, childrenOf map[string][]string, path string, seen map[string]bool) *db.FolderNode {
	name := filepath.Base(path)
	count := counts[path]

	children := childrenOf[path]
	node := &db.FolderNode{
		Path:      path,
		Name:      name,
		FileCount: count,
		Children:  make([]db.FolderNode, 0, len(children)),
	}

	seen[path] = true
	// Sort children by path so folders appear in alphabetical order
	slices.Sort(children)
	for _, childPath := range children {
		if seen[childPath] {
			continue // Skip cycles (e.g. folder listed as its own child)
		}
		node.Children = append(node.Children, *buildFolderTree(counts, childrenOf, childPath, seen))
	}

	return node
}

// resolveParent returns the parent of fPath, clamped to the vault root.
// fPath is a relative path (e.g. "photos/vacation"). The vault root is an
// absolute path (e.g. "C:\Vault").
//
// When filepath.Dir(fPath) yields "." (direct child of vault root) or
// resolves outside the vault, returns vaultPath as the root node.
func resolveParent(fPath string, vaultPath string) string {
	parentPath := filepath.Dir(fPath)

	// "fPath" is a relative path — if Dir returns "." it means the file
	// lives directly in the vault root.
	if parentPath == "." {
		return vaultPath
	}

	// For deeper relative paths (e.g. "photos/vacation" → "photos")
	// the parent is already a valid relative path — return it as-is.
	return parentPath
}

// AddExcludedFolder adds a folder to the exclusion list.
func (a *App) AddExcludedFolder(path string) error {
	v := a.vault()
	if v.db == nil {
		return fmt.Errorf("no vault open")
	}
	_, err := v.db.Conn().Exec(
		"INSERT INTO excluded_folders (path, created_at) VALUES (?, datetime('now'))",
		path,
	)
	return err
}

// RemoveExcludedFolder removes a folder from the exclusion list.
func (a *App) RemoveExcludedFolder(path string) error {
	v := a.vault()
	if v.db == nil {
		return fmt.Errorf("no vault open")
	}
	_, err := v.db.Conn().Exec(
		"DELETE FROM excluded_folders WHERE path = ?",
		path,
	)
	return err
}

// GetExcludedFolders returns the list of excluded folder paths.
func (a *App) GetExcludedFolders() ([]string, error) {
	v := a.vault()
	if v.db == nil {
		return nil, fmt.Errorf("no vault open")
	}
	rows, err := v.db.Conn().Query("SELECT path FROM excluded_folders")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var folders []string
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return nil, err
		}
		folders = append(folders, path)
	}
	return folders, nil
}

// OpenOriginalFile opens the original file with the default OS application.
func (a *App) OpenOriginalFile(fileID int64) error {
	v := a.vault()
	if v.db == nil {
		return fmt.Errorf("no vault open")
	}

	path, err := a.getFilePath(fileID)
	if err != nil {
		return err
	}

	_, err = os.Stat(path)
	if err != nil {
		return fmt.Errorf("file not found on disk: %w", err)
	}

	return utils.OpenWithDefaultApp(path)
}

// OpenFileFolder opens the parent folder of the file in the system file explorer.
func (a *App) OpenFileFolder(fileID int64) error {
	v := a.vault()
	if v.db == nil {
		return fmt.Errorf("no vault open")
	}

	path, err := a.getFilePath(fileID)
	if err != nil {
		return err
	}

	folder := filepath.Dir(path)
	_, err = os.Stat(folder)
	if err != nil {
		return fmt.Errorf("folder not found on disk: %w", err)
	}

	return utils.OpenFolder(folder)
}

// DeleteOriginalFile moves the original file to the recycle bin,
// removes its thumbnail, and removes the record from the vault index.
func (a *App) DeleteOriginalFile(fileID int64) error {
	v := a.vault()
	if v.db == nil {
		return fmt.Errorf("no vault open")
	}

	// Get file info first
	row := v.db.Conn().QueryRow(`
		SELECT vault_path, thumbnail_path FROM files WHERE id = ?
	`, fileID)

	var vaultPath string
	var thumbPath *string
	err := row.Scan(&vaultPath, &thumbPath)
	if err != nil {
		return fmt.Errorf("file not found in vault: %w", err)
	}

	// Resolve relative path to absolute for file operations
	absVaultPath := v.resolvePath(vaultPath)

	// Move original file to recycle bin
	err = utils.DeleteToTrash(absVaultPath)
	if err != nil {
		return fmt.Errorf("failed to delete original file: %w", err)
	}

	// Delete thumbnail from disk (if exists) — best-effort
	if thumbPath != nil && *thumbPath != "" {
		deleteThumbnailFile(v.resolvePath(*thumbPath))
	}

	// Remove from database (same as DeleteFile)
	tx, err := v.db.Conn().Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec("DELETE FROM file_tags WHERE file_id = ?", fileID)
	if err != nil {
		return fmt.Errorf("delete file_tags: %w", err)
	}

	res, err := tx.Exec("DELETE FROM files WHERE id = ?", fileID)
	if err != nil {
		return fmt.Errorf("delete file: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	if n, _ := res.RowsAffected(); n > 0 {
		a.adjustFileCount(-1)
	}
	return nil
}

// CopyImageToClipboard reads the original image file and places it on the
// system clipboard as a CF_DIB bitmap. Returns an error for non-image files
// or if the clipboard operation fails.
func (a *App) CopyImageToClipboard(fileID int64) error {
	v := a.vault()
	if v.db == nil {
		return fmt.Errorf("no vault open")
	}

	path, err := a.getFilePath(fileID)
	if err != nil {
		return err
	}

	_, err = os.Stat(path)
	if err != nil {
		return fmt.Errorf("file not found on disk: %w", err)
	}

	return utils.CopyImageToClipboard(path)
}

// getFilePath returns the absolute path for a given file ID.
// The vault_path in DB is relative; this resolves it to an absolute path.
func (a *App) getFilePath(fileID int64) (string, error) {
	v := a.vault()
	if v.db == nil {
		return "", fmt.Errorf("no vault open")
	}
	var relPath string
	err := v.db.Conn().QueryRow("SELECT vault_path FROM files WHERE id = ?", fileID).Scan(&relPath)
	if err != nil {
		return "", fmt.Errorf("file not found: %w", err)
	}
	return v.resolvePath(relPath), nil
}

// DeleteFile removes a file from the vault index (DB only, does NOT delete the actual file on disk).
func (a *App) DeleteFile(fileID int64) error {
	v := a.vault()
	if v.db == nil {
		return fmt.Errorf("no vault open")
	}

	// Remember the thumbnail location before the row is deleted so the
	// WebP can be removed from disk afterwards.
	var thumbRel *string
	_ = v.db.Conn().QueryRow("SELECT thumbnail_path FROM files WHERE id = ?", fileID).Scan(&thumbRel)

	tx, err := v.db.Conn().Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Remove tag associations
	_, err = tx.Exec("DELETE FROM file_tags WHERE file_id = ?", fileID)
	if err != nil {
		return fmt.Errorf("delete file_tags: %w", err)
	}

	// Remove from files table
	res, err := tx.Exec("DELETE FROM files WHERE id = ?", fileID)
	if err != nil {
		return fmt.Errorf("delete file: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	if n, _ := res.RowsAffected(); n > 0 {
		a.adjustFileCount(-1)
		// No row left → no thumbnail should be left either (the orphan
		// sweep in CleanupOrphanThumbnails would remove it anyway).
		if thumbRel != nil && *thumbRel != "" {
			deleteThumbnailFile(v.resolvePath(*thumbRel))
		}
	}
	return nil
}

// indexFile inserts a single file into the database.
func (a *App) indexFile(filePath string) error {
	if !utils.IsSupported(filePath) {
		return nil // Skip unsupported files
	}
	v := a.vault()
	if v.db == nil {
		return fmt.Errorf("no vault open")
	}

	folderPath := filepath.Dir(filePath)
	fileName := filepath.Base(filePath)
	now := time.Now().Format(time.RFC3339)
	info, statErr := os.Stat(filePath)
	if statErr != nil {
		return fmt.Errorf("failed to stat %s: %w", filePath, statErr)
	}
	ft := utils.GetFileTimes(info)

	createdAtStr := ""
	if !ft.CreatedAt.IsZero() {
		createdAtStr = ft.CreatedAt.Format(time.RFC3339)
	}

	relPath := v.toRelativePath(filePath)

	// Check before upserting whether the row already exists, so we know
	// afterwards whether the insert grew the files table.
	var existing int
	err := v.db.Conn().QueryRow(
		"SELECT 1 FROM files WHERE vault_path = ?", relPath,
	).Scan(&existing)
	isNew := err != nil // no row → not indexed yet

	_, err = v.db.Conn().Exec(`
		INSERT INTO files (vault_path, folder_path, filename, file_size, date_created, date_modified, indexed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(vault_path) DO UPDATE SET
			folder_path = excluded.folder_path,
			filename = excluded.filename,
			file_size = excluded.file_size,
			date_created = excluded.date_created,
			date_modified = excluded.date_modified,
			indexed_at = excluded.indexed_at
	`, relPath, v.toRelativePath(folderPath), fileName,
		info.Size(),
		createdAtStr,
		ft.ModifiedAt.Format(time.RFC3339),
		now)
	if err != nil {
		return err
	}

	if isNew {
		a.adjustFileCount(1)
	}
	return nil
}

// ImportResult holds the result of importing a single file.
type ImportResult struct {
	Imported int      `json:"imported"`
	Skipped  int      `json:"skipped"`
	Errors   []string `json:"errors"`
}

// ImportFile copies or moves a single file from outside the vault into the vault,
// optionally into a specific subfolder, indexes it in the database, and generates a thumbnail.
// If `move` is true, the source file is deleted after a successful copy.
// `targetFolder` is a path relative to the vault root; if empty, files go to vault root.
// Returns an ImportResult with counts of imported/skipped files and any errors.
func (a *App) ImportFile(sourcePath string, move bool, targetFolder string) *ImportResult {
	result := &ImportResult{Errors: []string{}}

	v := a.vault()
	if v.db == nil {
		result.Errors = append(result.Errors, "no vault open")
		return result
	}
	if v.path == "" {
		result.Errors = append(result.Errors, "no vault path set")
		return result
	}

	// Validate source file exists
	info, err := os.Stat(sourcePath)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("source file not found: %s", filepath.Base(sourcePath)))
		return result
	}
	if info.IsDir() {
		result.Errors = append(result.Errors, fmt.Sprintf("%s is a directory, not a file", filepath.Base(sourcePath)))
		return result
	}

	// Check if file type is supported
	if !utils.IsSupported(sourcePath) {
		result.Errors = append(result.Errors, fmt.Sprintf("unsupported file type: %s", filepath.Ext(sourcePath)))
		return result
	}

	// Resolve destination folder
	destDir := v.path
	if targetFolder != "" {
		destDir = filepath.Join(v.path, targetFolder)
		// Create target folder if it doesn't exist
		if _, err := os.Stat(destDir); os.IsNotExist(err) {
			if err = os.MkdirAll(destDir, 0755); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("failed to create folder %s: %v", targetFolder, err))
				return result
			}
		}
	}

	// Resolve destination path
	destPath := filepath.Join(destDir, info.Name())

	// Handle duplicate filenames — append (N) before extension
	if _, err := os.Stat(destPath); err == nil {
		base := strings.TrimSuffix(info.Name(), filepath.Ext(info.Name()))
		ext := filepath.Ext(info.Name())
		n := 2
		for {
			destPath = filepath.Join(destDir, fmt.Sprintf("%s (%d)%s", base, n, ext))
			if _, err := os.Stat(destPath); os.IsNotExist(err) {
				break
			}
			n++
		}
	}

	// Copy or move the file
	if move {
		err = os.Rename(sourcePath, destPath)
		// Rename across devices falls back to copy+delete
		if err != nil {
			err = copyFile(sourcePath, destPath)
			if err == nil {
				os.Remove(sourcePath)
			}
		}
	} else {
		err = copyFile(sourcePath, destPath)
	}
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("failed to %s file: %v", map[bool]string{true: "move", false: "copy"}[move], err))
		return result
	}

	// Check if already indexed (same absolute path)
	var existingID int64
	relPath := v.toRelativePath(destPath)
	err = v.db.Conn().QueryRow("SELECT id FROM files WHERE vault_path = ?", relPath).Scan(&existingID)
	if err == nil {
		// File already indexed — skip
		result.Skipped = 1
		return result
	}

	// Index the file
	folderPath := filepath.Dir(destPath)
	fileName := filepath.Base(destPath)
	now := time.Now().Format(time.RFC3339)
	ft := utils.GetFileTimes(info)

	createdAtStr := ""
	if !ft.CreatedAt.IsZero() {
		createdAtStr = ft.CreatedAt.Format(time.RFC3339)
	}

	_, err = v.db.Conn().Exec(`
		INSERT INTO files (vault_path, folder_path, filename, file_size, date_created, date_modified, indexed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, relPath, v.toRelativePath(folderPath), fileName,
		info.Size(),
		createdAtStr,
		ft.ModifiedAt.Format(time.RFC3339),
		now)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("failed to index file: %v", err))
		return result
	}

	// Plain INSERT after the existence check above — a new row was added.
	a.adjustFileCount(1)

	// Get the newly inserted file ID and generate thumbnail in background
	var newID int64
	err = v.db.Conn().QueryRow("SELECT id FROM files WHERE vault_path = ?", relPath).Scan(&newID)
	if err == nil {
		go func() {
			if _, err := a.GenerateThumbnail(newID); err != nil {
				fmt.Printf("thumbnail generation warning for %s: %v\n", fileName, err)
			}
		}()
	}

	result.Imported = 1
	return result
}

// copyFile copies a file from src to dst using buffered I/O.
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		os.Remove(dst) // Clean up partial copy
		return err
	}
	return dstFile.Close()
}
