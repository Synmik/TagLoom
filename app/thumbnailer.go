package app

import (
	"fmt"
	"os"
	"path/filepath"

	"TagLoom/utils"
)

// GenerateThumbnail creates a thumbnail for the given file ID.
// Thumbnails are stored in .tagloom/thumbnails/{2char_hash}/{hash}.jpg
func (a *App) GenerateThumbnail(fileID int64) (string, error) {
	if a.db == nil {
		return "", fmt.Errorf("no vault open")
	}

	// TODO: Fetch vault_path from DB
	// TODO: Generate thumbnail using imagemagick or gocv
	// TODO: Save to .tagloom/thumbnails/
	// TODO: Update thumbnail_path in DB
	return "", fmt.Errorf("not implemented")
}

// GenerateThumbnailPath returns the expected thumbnail path for a given file path.
func GenerateThumbnailPath(vaultPath, filePath string) string {
	hash := utils.HashPath(filePath)
	subdir := utils.ThumbnailSubdir(hash)
	thumbDir := filepath.Join(vaultPath, ".tagloom", "thumbnails", subdir)
	os.MkdirAll(thumbDir, 0755)
	return filepath.Join(thumbDir, hash+".jpg")
}

// CleanupOrphanThumbnails removes thumbnails for files no longer in the database.
func (a *App) CleanupOrphanThumbnails() error {
	// TODO: Compare .tagloom/thumbnails/ with DB records
	// TODO: Delete orphan files
	return fmt.Errorf("not implemented")
}
