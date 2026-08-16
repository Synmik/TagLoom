package app

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestThumbnailUpToDate(t *testing.T) {
	dir := t.TempDir()
	thumb := filepath.Join(dir, "thumb.webp")
	src := filepath.Join(dir, "src.png")
	if err := os.WriteFile(thumb, []byte("thumb"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(src, []byte("src"), 0644); err != nil {
		t.Fatal(err)
	}

	past := time.Now().Add(-2 * time.Hour)
	now := time.Now()
	future := time.Now().Add(time.Hour)

	// Thumbnail newer than source → up to date
	if err := os.Chtimes(src, past, past); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(thumb, now, now); err != nil {
		t.Fatal(err)
	}
	if !thumbnailUpToDate(thumb, src) {
		t.Error("expected up-to-date when source is older than thumbnail")
	}

	// Source newer than thumbnail → stale
	if err := os.Chtimes(src, future, future); err != nil {
		t.Fatal(err)
	}
	if thumbnailUpToDate(thumb, src) {
		t.Error("expected stale when source is newer than thumbnail")
	}

	// Missing thumbnail → not up to date (caller regenerates)
	if thumbnailUpToDate(filepath.Join(dir, "missing.webp"), src) {
		t.Error("expected not up-to-date when thumbnail is missing")
	}

	// Missing source → not up to date (caller regenerates / skips)
	if thumbnailUpToDate(thumb, filepath.Join(dir, "missing.png")) {
		t.Error("expected not up-to-date when source is missing")
	}
}
