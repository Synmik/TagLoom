package app

import (
	"fmt"
	"path/filepath"
	"time"

	"TagLoom/db"
	"TagLoom/utils"
)

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
	// TODO: Query files table grouped by folder_path
	// TODO: Build tree structure
	return nil, fmt.Errorf("not implemented")
}

// AddExcludedFolder adds a folder to the exclusion list.
func (a *App) AddExcludedFolder(path string) error {
	if a.db == nil {
		return fmt.Errorf("no vault open")
	}
	_, err := a.db.Conn().ExecContext(nil,
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
	_, err := a.db.Conn().ExecContext(nil,
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
