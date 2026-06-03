package app

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"TagLoom/db"
	"TagLoom/utils"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ScanVault performs a full re-scan of the vault directory.
// It discovers all supported media files and indexes them.
// Emits "scan:progress" events with {current, total} payload.
// Returns the total number of indexed files.
func (a *App) ScanVault() (int, error) {
	if a.db == nil {
		return 0, fmt.Errorf("no vault open")
	}
	if a.vaultPath == "" {
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

	// First pass: count total supported files for progress tracking
	var totalCount int
	err = filepath.WalkDir(a.vaultPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // Skip errors during count
		}
		if d.IsDir() {
			if isExcluded(path, excludedSet) {
				return filepath.SkipDir
			}
			return nil
		}
		if utils.IsSupported(path) {
			totalCount++
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("failed to walk directory: %w", err)
	}

	// Second pass: insert files into the database
	tx, err := a.db.Conn().Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}

	stmt, err := tx.Prepare(`
		INSERT INTO files (vault_path, folder_path, filename, date_modified, indexed_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(vault_path) DO UPDATE SET folder_path = excluded.folder_path, filename = excluded.filename, date_modified = excluded.date_modified, indexed_at = excluded.indexed_at
	`)
	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	var indexedCount int
	now := time.Now().Format(time.RFC3339)

	err = filepath.WalkDir(a.vaultPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if isExcluded(path, excludedSet) {
				return filepath.SkipDir
			}
			return nil
		}
		if !utils.IsSupported(path) {
			return nil
		}

		folderPath := filepath.Dir(path)
		fileName := filepath.Base(path)
		info, statErr := os.Stat(path)
		modStr := ""
		if statErr == nil {
			modStr = info.ModTime().Format(time.RFC3339)
		}
		_, execErr := stmt.Exec(path, folderPath, fileName, modStr, now)
		if execErr != nil {
			return fmt.Errorf("failed to insert %s: %w", path, execErr)
		}

		indexedCount++

		// Emit progress every 100 files
		if indexedCount%100 == 0 || indexedCount == totalCount {
			runtime.EventsEmit(a.ctx, "scan:progress", map[string]int{
				"current": indexedCount,
				"total":   totalCount,
			})
		}

		return nil
	})

	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("failed to scan vault: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Emit final progress
	runtime.EventsEmit(a.ctx, "scan:progress", map[string]int{
		"current": indexedCount,
		"total":   indexedCount,
	})
	runtime.EventsEmit(a.ctx, "scan:complete", indexedCount)

	return indexedCount, nil
}

// isExcluded checks if a directory path is in the excluded set.
// Always skips ".tagloom" (internal metadata directory) regardless of user config.
func isExcluded(path string, excluded map[string]bool) bool {
	clean := strings.ToLower(filepath.Clean(path))
	// Always exclude .tagloom — it contains thumbnails, DB, and config
	base := filepath.Base(clean)
	if base == ".tagloom" {
		return true
	}
	for excl := range excluded {
		if clean == excl || strings.HasPrefix(clean, excl+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

// RescanVault performs a diff scan, detecting added and removed files.
// Returns the number of added files. Removed count is sent via rescan:complete event.
func (a *App) RescanVault() (int, error) {
	if a.db == nil {
		return 0, fmt.Errorf("no vault open")
	}
	if a.vaultPath == "" {
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

	// Step 1: Collect all files from the filesystem
	fsFiles := make(map[string]bool)
	var fsPaths []string
	err = filepath.WalkDir(a.vaultPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if isExcluded(path, excludedSet) {
				return filepath.SkipDir
			}
			return nil
		}
		if utils.IsSupported(path) {
			fsFiles[path] = true
			fsPaths = append(fsPaths, path)
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("failed to walk directory: %w", err)
	}

	// Step 2: Collect all files from the DB
	rows, err := a.db.Conn().Query("SELECT vault_path FROM files")
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

	// Step 3: Compute diff
	var added []string
	for _, p := range fsPaths {
		if !dbFiles[p] {
			added = append(added, p)
		}
	}

	var removed []string
	for p := range dbFiles {
		if !fsFiles[p] {
			removed = append(removed, p)
		}
	}

	// Emit diff summary
	runtime.EventsEmit(a.ctx, "rescan:diff", map[string]int{
		"added":   len(added),
		"removed": len(removed),
		"total":   len(fsFiles),
	})

	// Step 4: Insert new files in a transaction
	tx, err := a.db.Conn().Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Insert new files
	if len(added) > 0 {
		stmt, err := tx.Prepare(`
			INSERT INTO files (vault_path, folder_path, filename, date_created, date_modified, indexed_at)
			VALUES (?, ?, ?, ?, ?, ?)
		`)
		if err != nil {
			tx.Rollback()
			return 0, fmt.Errorf("failed to prepare insert: %w", err)
		}

		now := time.Now().Format(time.RFC3339)
		for i, path := range added {
			info, statErr := os.Stat(path)
			fileName := filepath.Base(path)
			modStr := ""
			if statErr == nil {
				modStr = info.ModTime().Format(time.RFC3339)
			}
			_, err := stmt.Exec(path, filepath.Dir(path), fileName, modStr, now)
			if err != nil {
				stmt.Close()
				tx.Rollback()
				return 0, fmt.Errorf("failed to insert %s: %w", path, err)
			}
			// Emit progress frequently enough for small batches too
			step := (i + 1) * 100 / len(added)
			if step%25 == 0 || i+1 == len(added) {
				runtime.EventsEmit(a.ctx, "rescan:progress", map[string]interface{}{
					"phase":   "adding",
					"current": i + 1,
					"total":   len(added),
				})
			}
		}
		stmt.Close()
	}

	// Delete removed files (and their tags)
	if len(removed) > 0 {
		delStmt, err := tx.Prepare("DELETE FROM file_tags WHERE file_id = ?")
		if err == nil {
			for _, p := range removed {
				delStmt.Exec(p) // ignore errors — cascade will handle it
			}
			delStmt.Close()
		}

		delFileStmt, err := tx.Prepare("DELETE FROM files WHERE vault_path = ?")
		if err != nil {
			tx.Rollback()
			return 0, fmt.Errorf("failed to prepare delete: %w", err)
		}

			for i, p := range removed {
			_, err := delFileStmt.Exec(p)
			if err != nil {
				delFileStmt.Close()
				tx.Rollback()
				return 0, fmt.Errorf("failed to delete %s: %w", p, err)
			}
			step := (i + 1) * 100 / len(removed)
			if step%25 == 0 || i+1 == len(removed) {
				runtime.EventsEmit(a.ctx, "rescan:progress", map[string]interface{}{
					"phase":   "removing",
					"current": i + 1,
					"total":   len(removed),
				})
			}
		}
		delFileStmt.Close()
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit: %w", err)
	}

	// Emit completion
	runtime.EventsEmit(a.ctx, "rescan:complete", map[string]int{
		"added":   len(added),
		"removed": len(removed),
	})

	return len(added), nil
}

// GetFolderTree returns the recursive folder tree for the vault.
func (a *App) GetFolderTree(path string) *db.FolderNode {
	if a.db == nil {
		return &db.FolderNode{
			Path:     path,
			Name:     filepath.Base(path),
			FileCount: 0,
			Children: []db.FolderNode{},
		}
	}

	// Query all unique folder paths with file counts
	rows, err := a.db.Conn().Query(`
		SELECT folder_path, COUNT(*) as cnt
		FROM files
		GROUP BY folder_path
		ORDER BY folder_path
	`)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to query folder tree: %v\n", err)
		return &db.FolderNode{
			Path:     path,
			Name:     filepath.Base(path),
			FileCount: 0,
			Children: []db.FolderNode{},
		}
	}
	defer rows.Close()

	// Build a map of folder_path → file count
	folderCounts := make(map[string]int)
	for rows.Next() {
		var fPath string
		var count int
		if err := rows.Scan(&fPath, &count); err != nil {
			continue
		}
		folderCounts[fPath] = count
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
// If filepath.Dir(fPath) is outside the vault (e.g. the drive root), returns vaultPath.
// Also guards against self-parent (e.g. drive roots where Dir(x) == x).
func resolveParent(fPath string, vaultPath string) string {
	parentPath := filepath.Dir(fPath)
	rel, err := filepath.Rel(vaultPath, parentPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return vaultPath
	}
	return parentPath
}

// AddExcludedFolder adds a folder to the exclusion list.
func (a *App) AddExcludedFolder(path string) error {
	if a.db == nil {
		return fmt.Errorf("no vault open")
	}
	_, err := a.db.Conn().Exec(
		"INSERT INTO excluded_folders (path, created_at) VALUES (?, datetime('now'))",
		path,
	)
	return err
}

// RemoveExcludedFolder removes a folder from the exclusion list.
func (a *App) RemoveExcludedFolder(path string) error {
	if a.db == nil {
		return fmt.Errorf("no vault open")
	}
	_, err := a.db.Conn().Exec(
		"DELETE FROM excluded_folders WHERE path = ?",
		path,
	)
	return err
}

// GetExcludedFolders returns the list of excluded folder paths.
func (a *App) GetExcludedFolders() ([]string, error) {
	if a.db == nil {
		return nil, fmt.Errorf("no vault open")
	}
	rows, err := a.db.Conn().Query("SELECT path FROM excluded_folders")
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

// DeleteFile removes a file from the vault index (DB only, does NOT delete the actual file on disk).
func (a *App) DeleteFile(fileID int64) error {
	if a.db == nil {
		return fmt.Errorf("no vault open")
	}

	tx, err := a.db.Conn().Begin()
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
	_, err = tx.Exec("DELETE FROM files WHERE id = ?", fileID)
	if err != nil {
		return fmt.Errorf("delete file: %w", err)
	}

	return tx.Commit()
}

// indexFile inserts a single file into the database.
func (a *App) indexFile(filePath string) error {
	if !utils.IsSupported(filePath) {
		return nil // Skip unsupported files
	}

	folderPath := filepath.Dir(filePath)
	fileName := filepath.Base(filePath)
	now := time.Now().Format(time.RFC3339)
	info, statErr := os.Stat(filePath)
	modStr := ""
	if statErr == nil {
		modStr = info.ModTime().Format(time.RFC3339)
	}

	_, err := a.db.Conn().Exec(`
		INSERT INTO files (vault_path, folder_path, filename, date_modified, indexed_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(vault_path) DO UPDATE SET folder_path = excluded.folder_path, filename = excluded.filename, date_modified = excluded.date_modified, indexed_at = excluded.indexed_at
	`, filePath, folderPath, fileName, modStr, now)

	return err
}
