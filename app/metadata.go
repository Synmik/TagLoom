package app

import (
	"fmt"
	"os"
	"path/filepath"

	"TagLoom/utils"
)

// FileMetadata holds read-only metadata extracted from the original file.
// This is fetched on-demand when the metadata panel is opened.
type FileMetadata struct {
	Filename         string  `json:"filename"`
	Extension        string  `json:"extension"`
	FormatName       string  `json:"format_name"`
	MimeType         string  `json:"mime_type"`
	SizeBytes        int64   `json:"size_bytes"`
	DateCreated      string  `json:"date_created"`
	DateModified     string  `json:"date_modified"`
	ResolutionWidth  int     `json:"resolution_width"`
	ResolutionHeight int     `json:"resolution_height"`
	DurationSeconds  float64 `json:"duration_seconds"`
	DominantColors   []string `json:"dominant_colors"`
}

// GetFileMetadata extracts metadata from the original file on disk.
// This is called when the user opens the metadata panel for a file.
func (a *App) GetFileMetadata(fileID int64) (*FileMetadata, error) {
	if a.db == nil {
		return nil, fmt.Errorf("no vault open")
	}

	// Fetch vault_path from DB
	file, err := a.GetFileByID(fileID)
	if err != nil {
		return nil, fmt.Errorf("file not found: %w", err)
	}

	// Check if file still exists
	info, err := os.Stat(file.VaultPath)
	if err != nil {
		return nil, fmt.Errorf("file not accessible: %w", err)
	}

	ext := filepath.Ext(file.VaultPath)
	category := utils.GetFileCategory(file.VaultPath)

	metadata := &FileMetadata{
		Filename:         utils.GetFilename(file.VaultPath),
		Extension:        ext,
		FormatName:       utils.GetFormatName(file.VaultPath),
		SizeBytes:        info.Size(),
		DateCreated:      info.ModTime().Format("2006-01-02 15:04:05"), // TODO: Get actual creation time on Windows
		DateModified:     info.ModTime().Format("2006-01-02 15:04:05"),
	}

	// TODO: Extract resolution using exif library for images
	// TODO: Extract duration using go-media for videos
	// TODO: Extract dominant colors using color palette algorithm
	// TODO: Set mime_type based on extension

	_ = category // Used in future MIME type mapping

	return metadata, nil
}

// GetBatchMetadata extracts lightweight metadata for a batch of files.
// Used for gallery rendering (filename, format badge) and sorting.
func (a *App) GetBatchMetadata(fileIDs []int64) ([]FileMetadata, error) {
	// TODO: Implement batch metadata extraction
	// TODO: Cache results in Vue per-page
	return nil, fmt.Errorf("not implemented")
}
