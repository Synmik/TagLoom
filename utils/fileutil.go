package utils

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rwcarlsen/goexif/exif"
)

// SupportedExtensions maps file categories to their extensions.
var SupportedExtensions = map[string][]string{
	"image": {".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".tiff", ".tif", ".svg"},
	"video": {".mp4", ".mov", ".avi", ".webm", ".mkv", ".wmv", ".flv"},
	"animated": {".gif", ".webm"},
}

// IsSupported checks if a file extension is supported.
func IsSupported(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	for _, exts := range SupportedExtensions {
		for _, e := range exts {
			if ext == e {
				return true
			}
		}
	}
	return false
}

// GetFileCategory returns the category of a file based on its extension.
func GetFileCategory(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	for category, exts := range SupportedExtensions {
		for _, e := range exts {
			if ext == e {
				return category
			}
		}
	}
	return "unknown"
}

// GetFormatName returns a human-readable format name from the extension.
func GetFormatName(path string) string {
	ext := strings.ToUpper(strings.TrimPrefix(filepath.Ext(path), "."))
	if ext == "JPG" || ext == "JPEG" {
		return "JPEG"
	}
	if ext == "TIF" || ext == "TIFF" {
		return "TIFF"
	}
	return ext
}

// GetFilename extracts the filename from a full path.
func GetFilename(path string) string {
	return filepath.Base(path)
}

// GetCreationTimeNanos returns the file creation time from EXIF metadata.
// Returns 0 if no EXIF data is available.
func GetCreationTimeNanos(path string) int64 {
	return getEXIFCreationTime(path)
}

// getEXIFCreationTime extracts the original creation time from EXIF metadata.
// Returns Unix nanoseconds, or 0 if not available.
func getEXIFCreationTime(path string) int64 {
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close()

	exifReader, err := exif.Decode(f)
	if err != nil {
		return 0
	}

	// Try tags in priority order: DateTimeOriginal, DateTimeDigitized, DateTime
	for _, tagName := range []string{"DateTimeOriginal", "DateTimeDigitized", "DateTime"} {
		tagVal, err := exifReader.Get(exif.FieldName(tagName))
		if err != nil {
			continue
		}
		s, err := tagVal.StringVal()
		if err != nil || s == "" {
			continue
		}
		// EXIF date format: "2006:01:02 15:04:05"
		t, err := time.Parse("2006:01:02 15:04:05", s)
		if err != nil {
			continue
		}
		return t.UnixNano()
	}
	return 0
}
