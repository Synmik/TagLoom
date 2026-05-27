package utils

import (
	"crypto/sha256"
	"fmt"
)

// HashPath generates a deterministic hash for a file path.
// Used to create unique thumbnail filenames.
func HashPath(path string) string {
	hash := sha256.Sum256([]byte(path))
	return fmt.Sprintf("%x", hash[:16]) // 32 hex chars
}

// ThumbnailSubdir returns the 2-char prefix subdirectory for a hash.
// This prevents a single directory from having too many files on Windows.
func ThumbnailSubdir(hash string) string {
	if len(hash) >= 2 {
		return hash[:2]
	}
	return "00"
}
