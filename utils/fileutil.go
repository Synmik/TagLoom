package utils

import (
	"path/filepath"
	"strings"
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
