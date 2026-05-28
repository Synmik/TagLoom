package app

import (
	"fmt"
	"path/filepath"
	"time"

	"TagLoom/db"
	"TagLoom/utils"
)

// folderInfo holds a folder path and its file count for tree building.
type folderInfo struct {
	path  string
	count int
}

// ScanVault performs a full re-scan of the vault directory.
// It discovers all supported media files and indexes them.
func (a *App) ScanVault(progress chan int) error {
	if a.db == nil {
		return fmt.Errorf("no vault open")
	}
	// TODO: Implement folder tree walk
	// TODO: For each supported file, insert into files table
	// TODO: Send progress updates via channel
	return fmt.Errorf("not implemented")
}

// RescanVault performs a diff scan, detecting added and removed files.
func (a *App) RescanVault(progress chan int) error {
	if a.db == nil {
		return fmt.Errorf("no vault open")
	}
	// TODO: Compare filesystem with DB records
	// TODO: Insert new files, mark deleted ones
	return fmt.Errorf("not implemented")
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
		INSERT OR REPLACE INTO files (vault_path, folder_path, indexed_at)
		VALUES (?, ?, ?)
	`, filePath, folderPath, now)

	return err
}
