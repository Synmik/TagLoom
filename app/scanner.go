package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"TagLoom/db"
	"TagLoom/utils"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// folderInfo holds a folder path and its file count for tree building.
type folderInfo struct {
	path  string
	count int
}

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
		INSERT INTO files (vault_path, folder_path, indexed_at)
		VALUES (?, ?, ?)
		ON CONFLICT(vault_path) DO UPDATE SET folder_path = excluded.folder_path, indexed_at = excluded.indexed_at
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
		_, execErr := stmt.Exec(path, folderPath, now)
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
			INSERT INTO files (vault_path, folder_path, indexed_at)
			VALUES (?, ?, ?)
		`)
		if err != nil {
			tx.Rollback()
			return 0, fmt.Errorf("failed to prepare insert: %w", err)
		}

		now := time.Now().Format(time.RFC3339)
		for i, path := range added {
			_, err := stmt.Exec(path, filepath.Dir(path), now)
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
func (a *App) GetFolderTree(path string) (*db.FolderNode, error) {
	if a.db == nil {
		return nil, fmt.Errorf("no vault open")
	}

	// Query all unique folder paths with file counts
	rows, err := a.db.Conn().Query(`
		SELECT folder_path, COUNT(*) as cnt
		FROM files
		GROUP BY folder_path
		ORDER BY folder_path
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query folder tree: %w", err)
	}
	defer rows.Close()

	// Build a map of folder_path → node
	var folders []folderInfo
	for rows.Next() {
		var fi folderInfo
		if err := rows.Scan(&fi.path, &fi.count); err != nil {
			return nil, err
		}
		folders = append(folders, fi)
	}

	// Build tree from flat list
	return buildFolderTree(folders, path), nil
}

// buildFolderTree constructs a recursive tree from a flat list of folder paths.
func buildFolderTree(folders []folderInfo, rootPath string) *db.FolderNode {
	// Collect unique folder paths with counts
	folderMap := make(map[string]*db.FolderNode)

	for _, fi := range folders {
		// Normalize path separators
		normalized := filepath.ToSlash(fi.path)
		name := filepath.Base(normalized)
		if name == "" {
			name = filepath.Base(fi.path)
		}

		node, exists := folderMap[fi.path]
		if !exists {
			node = &db.FolderNode{
				Path:      fi.path,
				Name:      name,
				FileCount: fi.count,
			}
			folderMap[fi.path] = node
		}
	}

	// Build parent-child relationships
	var root *db.FolderNode
	for _, node := range folderMap {
		parentPath := filepath.Dir(node.Path)
		// Check if parent is in our map
		parent, hasParent := folderMap[parentPath]
		if hasParent {
			parent.Children = append(parent.Children, *node)
		} else {
			// Top-level folder
			if root == nil {
				root = &db.FolderNode{
					Path:      rootPath,
					Name:      filepath.Base(rootPath),
					FileCount: 0,
				}
			}
			root.Children = append(root.Children, *node)
		}
	}

	if root == nil {
		root = &db.FolderNode{
			Path:      rootPath,
			Name:      filepath.Base(rootPath),
			FileCount: 0,
		}
	}

	return root
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

// indexFile inserts a single file into the database.
func (a *App) indexFile(filePath string) error {
	if !utils.IsSupported(filePath) {
		return nil // Skip unsupported files
	}

	folderPath := filepath.Dir(filePath)
	now := time.Now().Format(time.RFC3339)

	_, err := a.db.Conn().Exec(`
		INSERT INTO files (vault_path, folder_path, indexed_at)
		VALUES (?, ?, ?)
		ON CONFLICT(vault_path) DO UPDATE SET folder_path = excluded.folder_path, indexed_at = excluded.indexed_at
	`, filePath, folderPath, now)

	return err
}
