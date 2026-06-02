//go:build windows

package app

import (
	"encoding/binary"
	"fmt"
	"image"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"TagLoom/utils"

	"github.com/rwcarlsen/goexif/exif"
)

// FileMetadata holds read-only metadata extracted from the original file.
// This is fetched on-demand when the metadata panel is opened.
type FileMetadata struct {
	Filename         string   `json:"filename"`
	Extension        string   `json:"extension"`
	FormatName       string   `json:"format_name"`
	MimeType         string   `json:"mime_type"`
	SizeBytes        int64    `json:"size_bytes"`
	DateCreated      string   `json:"date_created"`
	DateModified     string   `json:"date_modified"`
	ResolutionWidth  int      `json:"resolution_width"`
	ResolutionHeight int      `json:"resolution_height"`
	DurationSeconds  float64  `json:"duration_seconds"`
	DominantColors   []string `json:"dominant_colors"`
}

// mimeTypeMap maps file extensions to MIME types.
var mimeTypeMap = map[string]string{
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".gif":  "image/gif",
	".webp": "image/webp",
	".bmp":  "image/bmp",
	".tiff": "image/tiff",
	".tif":  "image/tiff",
	".svg":  "image/svg+xml",
	".mp4":  "video/mp4",
	".mov":  "video/quicktime",
	".avi":  "video/x-msvideo",
	".webm": "video/webm",
	".mkv":  "video/x-matroska",
	".wmv":  "video/x-ms-wmv",
	".flv":  "video/x-flv",
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

	ext := strings.ToLower(filepath.Ext(file.VaultPath))
	category := utils.GetFileCategory(file.VaultPath)

	metadata := &FileMetadata{
		Filename:     utils.GetFilename(file.VaultPath),
		Extension:    ext,
		FormatName:   utils.GetFormatName(file.VaultPath),
		MimeType:     mimeTypeMap[ext],
		SizeBytes:    info.Size(),
		DateModified: info.ModTime().Format("2006-01-02 15:04:05"),
	}

	// Get file creation time from Windows
	if created := getFileCreationTime(file.VaultPath); created != "" {
		metadata.DateCreated = created
	} else {
		metadata.DateCreated = metadata.DateModified
	}

	// Extract resolution and duration based on file category
	switch category {
	case "image", "animated":
		w, h, err := getImageDimensions(file.VaultPath, ext)
		if err == nil {
			metadata.ResolutionWidth = w
			metadata.ResolutionHeight = h
		}
		// Try EXIF for additional data
		if exifData, exifErr := extractEXIF(file.VaultPath); exifErr == nil {
			if w == 0 && h == 0 {
				if exifData.Width > 0 && exifData.Height > 0 {
					metadata.ResolutionWidth = exifData.Width
					metadata.ResolutionHeight = exifData.Height
				}
			}
		}
	case "video":
		// Use ffprobe for all video metadata (duration, resolution, codec)
		if probeInfo, probeErr := utils.ProbeVideo(file.VaultPath); probeErr == nil {
			metadata.DurationSeconds = probeInfo.DurationSeconds
			metadata.ResolutionWidth = probeInfo.Width
			metadata.ResolutionHeight = probeInfo.Height
		} else {
			// Fallback to legacy moov parsing for MP4/MOV
			if d, err := getVideoDuration(file.VaultPath, ext); err == nil {
				metadata.DurationSeconds = d
			}
		}
	}

	return metadata, nil
}

// getImageDimensions returns image width and height without loading the full image.
func getImageDimensions(path string, ext string) (int, int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()

	config, _, err := image.DecodeConfig(f)
	if err != nil {
		return 0, 0, err
	}
	return int(config.Width), int(config.Height), nil
}

// exifData holds EXIF resolution data.
type exifData struct {
	Width  int
	Height int
}

// extractEXIF reads EXIF data from an image file.
func extractEXIF(path string) (exifData, error) {
	f, err := os.Open(path)
	if err != nil {
		return exifData{}, err
	}
	defer f.Close()

	exifReader, err := exif.Decode(f)
	if err != nil {
		return exifData{}, err
	}

	var data exifData
	if tag, err := exifReader.Get(exif.PixelXDimension); err == nil {
		if val, err := tag.Int(0); err == nil {
			data.Width = int(val)
		}
	}
	if tag, err := exifReader.Get(exif.PixelYDimension); err == nil {
		if val, err := tag.Int(0); err == nil {
			data.Height = int(val)
		}
	}
	return data, nil
}

// getVideoDuration extracts duration from MP4/MOV files by parsing the moov atom.
func getVideoDuration(path string, ext string) (float64, error) {
	switch ext {
	case ".mp4", ".mov":
		return getMP4Duration(path)
	case ".webm", ".mkv", ".avi", ".wmv", ".flv":
		// These formats require more complex parsing or FFmpeg
		return 0, fmt.Errorf("duration extraction not supported for %s", ext)
	default:
		return 0, fmt.Errorf("unknown video format: %s", ext)
	}
}

// getMP4Duration parses the moov atom to extract duration.
func getMP4Duration(path string) (float64, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	// Find the moov atom
	buf := make([]byte, 12) // 8 bytes header + 4 bytes for nested atom
	for {
		_, err := io.ReadFull(f, buf)
		if err != nil {
			break
		}

		size := int(binary.BigEndian.Uint32(buf[0:4]))
		atomType := string(buf[4:8])

		if atomType == "moov" {
			// Found moov, now search for mvhd inside it
			moovSize := size
			moovRead := 8
			for moovRead < moovSize {
				remaining := moovSize - moovRead
				if remaining < 8 {
					break
				}

				// Read next chunk
				chunkSize := min(remaining, len(buf))
				_, err := io.ReadFull(f, buf[:chunkSize])
				if err != nil {
					break
				}
				moovRead += chunkSize

				if chunkSize < 12 {
					continue
				}

				nestedSize := int(binary.BigEndian.Uint32(buf[0:4]))
				nestedType := string(buf[4:8])

				if nestedType == "mvhd" {
					// mvhd version 1 has duration at offset 20 (8 bytes)
					// mvhd version 0 has duration at offset 12 (4 bytes)
					version := buf[8]
					if version == 1 && chunkSize >= 28 {
						// 64-bit duration
						durationHi := binary.BigEndian.Uint32(buf[20:24])
						_, err := io.ReadFull(f, buf[:4])
						if err != nil {
							break
						}
						durationLo := binary.BigEndian.Uint32(buf[0:4])
						duration := (uint64(durationHi) << 32) | uint64(durationLo)
						// Timescale is at offset 12-16
						_, err = f.Seek(-(int64(chunkSize) - 4), io.SeekCurrent)
						if err != nil {
							break
						}
						_, err = io.ReadFull(f, buf[:4])
						if err != nil {
							break
						}
						timescale := binary.BigEndian.Uint32(buf[0:4])
						if timescale > 0 {
							return float64(duration) / float64(timescale), nil
						}
					} else if chunkSize >= 16 {
						// 32-bit duration
						duration := binary.BigEndian.Uint32(buf[12:16])
						timescale := binary.BigEndian.Uint32(buf[16:20])
						if timescale > 0 {
							return float64(duration) / float64(timescale), nil
						}
					}
					break
				}

				// Skip to next atom in moov
				skip := max(nestedSize-8, 1)
				if _, err := io.CopyN(io.Discard, f, int64(skip)); err != nil {
					break
				}
				moovRead += skip
			}
			return 0, fmt.Errorf("could not parse moov atom")
		}

		// Skip this atom
		if size > 8 {
			if _, err := io.CopyN(io.Discard, f, int64(size-8)); err != nil {
				break
			}
		}
	}

	return 0, fmt.Errorf("moov atom not found")
}

// getVideoDimensions tries to get video resolution from container headers.
// This is a simplified version that works for MP4/MOV.
func getVideoDimensions(path string, ext string) (int, int, error) {
	// For now, return 0 - full video dimension extraction requires FFmpeg
	return 0, 0, fmt.Errorf("video dimensions require FFmpeg")
}

// getFileCreationTime extracts the original creation time.
// Priority: EXIF DateTimeOriginal > Windows CreationTime > ModTime.
func getFileCreationTime(path string) string {
	// Try EXIF first (for JPEG, TIFF, PNG, WebP)
	if nanos := utils.GetCreationTimeNanos(path); nanos > 0 {
		return time.Unix(0, nanos).Format("2006-01-02 15:04:05")
	}

	// Try Windows CreationTime
	if t := getWindowsCreationTime(path); t != "" {
		return t
	}

	// Last resort: ModTime
	info, err := os.Stat(path)
	if err != nil {
		return ""
	}

	return info.ModTime().Format("2006-01-02 15:04:05")
}

// getWindowsCreationTime reads the file creation time from Windows.
func getWindowsCreationTime(path string) string {
	info, err := os.Stat(path)
	if err != nil {
		return ""
	}
	sys := info.Sys()
	if win, ok := sys.(*syscall.Win32FileAttributeData); ok {
		return time.Unix(0, win.CreationTime.Nanoseconds()).Format("2006-01-02 15:04:05")
	}
	return ""
}

// GetBatchMetadata extracts lightweight metadata for a batch of files.
// Used for gallery rendering (filename, format badge) and sorting.
func (a *App) GetBatchMetadata(fileIDs []int64) ([]FileMetadata, error) {
	if a.db == nil {
		return nil, fmt.Errorf("no vault open")
	}

	var results []FileMetadata
	for _, id := range fileIDs {
		meta, err := a.GetFileMetadata(id)
		if err != nil {
			continue // Skip files that fail
		}
		results = append(results, *meta)
	}
	return results, nil
}

// Helper functions for min/max (Go 1.21+)
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Ensure time is used
var _ = time.Now
