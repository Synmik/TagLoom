package app

import (
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	"os"
	"path/filepath"
	"strings"

	"TagLoom/utils"

	"github.com/disintegration/imaging"
	"golang.org/x/image/tiff"
	"golang.org/x/image/webp"
)

const defaultThumbnailSize = 256
const defaultThumbnailQuality = 80

// GenerateThumbnail creates a thumbnail for the given file ID.
// Thumbnails are stored in .tagloom/thumbnails/{2char_hash}/{hash}.jpg
func (a *App) GenerateThumbnail(fileID int64) (string, error) {
	if a.db == nil {
		return "", fmt.Errorf("no vault open")
	}
	if a.vaultPath == "" {
		return "", fmt.Errorf("no vault path set")
	}

	// Fetch file info from DB
	row := a.db.Conn().QueryRow(`
		SELECT id, vault_path, thumbnail_path
		FROM files WHERE id = ?
	`, fileID)

	var id int64
	var vaultPath, existingThumb string
	if err := row.Scan(&id, &vaultPath, &existingThumb); err != nil {
		return "", fmt.Errorf("failed to find file %d: %w", fileID, err)
	}

	// Check if file actually exists on disk
	if _, err := os.Stat(vaultPath); err != nil {
		return "", fmt.Errorf("file not found on disk: %w", err)
	}

	// Determine target thumbnail path
	thumbPath := GenerateThumbnailPath(a.vaultPath, vaultPath)

	// If thumbnail already exists, check if source is newer
	if existingThumb == thumbPath {
		thumbStat, err1 := os.Stat(thumbPath)
		srcStat, err2 := os.Stat(vaultPath)
		if err1 == nil && err2 == nil && !srcStat.ModTime().After(thumbStat.ModTime()) {
			return thumbPath, nil // Thumbnail is up to date
		}
	}

	// Determine file category
	ext := strings.ToLower(filepath.Ext(vaultPath))
	category := utils.GetFileCategory(vaultPath)

	var img image.Image
	var err error

	switch category {
	case "image", "animated":
		img, err = decodeImage(vaultPath, ext)
		if err != nil {
			return "", fmt.Errorf("failed to decode image: %w", err)
		}
	default:
		// Video thumbnails require FFmpeg — skip for now
		return "", fmt.Errorf("thumbnail generation not supported for %s files yet", ext)
	}

	// Get config for thumbnail size/quality
	size := defaultThumbnailSize
	quality := defaultThumbnailQuality
	if a.vaultCfg != nil {
		if a.vaultCfg.Settings.ThumbnailSize > 0 {
			size = a.vaultCfg.Settings.ThumbnailSize
		}
		if a.vaultCfg.Settings.ThumbnailQuality > 0 {
			quality = a.vaultCfg.Settings.ThumbnailQuality
		}
	}

	// Resize to thumbnail (fit within size×size, maintain aspect ratio)
	resized := imaging.Fit(img, size, size, imaging.Lanczos)

	// Ensure output directory exists
	thumbDir := filepath.Dir(thumbPath)
	if err := os.MkdirAll(thumbDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create thumbnail directory: %w", err)
	}

	// Write as JPEG
	outFile, err := os.Create(thumbPath)
	if err != nil {
		return "", fmt.Errorf("failed to create thumbnail file: %w", err)
	}
	defer outFile.Close()

	if err := jpeg.Encode(outFile, resized, &jpeg.Options{Quality: quality}); err != nil {
		outFile.Close()
		os.Remove(thumbPath) // Clean up partial file
		return "", fmt.Errorf("failed to encode JPEG: %w", err)
	}

	// Update DB with thumbnail path
	_, err = a.db.Conn().Exec("UPDATE files SET thumbnail_path = ? WHERE id = ?", thumbPath, fileID)
	if err != nil {
		return "", fmt.Errorf("failed to update thumbnail_path: %w", err)
	}

	return thumbPath, nil
}

// decodeImage opens and decodes an image file, supporting multiple formats.
func decodeImage(path string, ext string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// For WebP and TIFF we need explicit decoders; stdlib handles the rest.
	switch ext {
	case ".webp":
		return webp.Decode(f)
	case ".tiff", ".tif":
		return tiff.Decode(f)
	default:
		// jpeg, png, gif, bmp — all handled by image.Decode
		img, _, err := image.Decode(f)
		return img, err
	}
}

// GenerateThumbnailPath returns the expected thumbnail path for a given file path.
func GenerateThumbnailPath(vaultPath, filePath string) string {
	hash := utils.HashPath(filePath)
	subdir := utils.ThumbnailSubdir(hash)
	thumbDir := filepath.Join(vaultPath, ".tagloom", "thumbnails", subdir)
	os.MkdirAll(thumbDir, 0755)
	return filepath.Join(thumbDir, hash+".jpg")
}

// GenerateThumbnailsForFiles generates thumbnails for a list of file IDs.
// Returns the number of thumbnails generated.
func (a *App) GenerateThumbnailsForFiles(fileIDs []int64) int {
	count := 0
	for _, id := range fileIDs {
		_, err := a.GenerateThumbnail(id)
		if err == nil {
			count++
		}
	}
	return count
}

// CleanupOrphanThumbnails removes thumbnails for files no longer in the database.
func (a *App) CleanupOrphanThumbnails() error {
	// TODO: Compare .tagloom/thumbnails/ with DB records
	// TODO: Delete orphan files
	return fmt.Errorf("not implemented")
}

// GetThumbnailData reads a thumbnail file and returns it as a base64 data URL.
// The frontend uses this to display thumbnails in <img> tags.
func (a *App) GetThumbnailData(fileID int64) (string, error) {
	if a.db == nil {
		return "", fmt.Errorf("no vault open")
	}

	// Fetch thumbnail_path from DB
	row := a.db.Conn().QueryRow("SELECT thumbnail_path FROM files WHERE id = ?", fileID)
	var thumbPath string
	if err := row.Scan(&thumbPath); err != nil {
		return "", fmt.Errorf("no thumbnail for file %d: %w", fileID, err)
	}
	if thumbPath == "" {
		return "", fmt.Errorf("no thumbnail path for file %d", fileID)
	}

	data, err := os.ReadFile(thumbPath)
	if err != nil {
		return "", fmt.Errorf("failed to read thumbnail: %w", err)
	}

	// Encode as base64 data URL
	b64 := base64.StdEncoding.EncodeToString(data)
	return "data:image/jpeg;base64," + b64, nil
}

// ServeThumbnail reads a thumbnail file and returns its bytes.
// Used by the frontend to display thumbnails via a Go handler.
func ServeThumbnail(thumbPath string) ([]byte, error) {
	if thumbPath == "" {
		return nil, fmt.Errorf("no thumbnail path")
	}
	data, err := os.ReadFile(thumbPath)
	if err != nil {
		return nil, err
	}
	return data, nil
}
