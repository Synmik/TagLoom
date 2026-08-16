package app

import (
	"image"
	"image/png"
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

// writeTestPNG creates a small valid PNG on disk (real image bytes, so
// ffmpeg can encode it to WebP).
func writeTestPNG(t *testing.T, path string) {
	t.Helper()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create %s: %v", path, err)
	}
	defer f.Close()
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	if err := png.Encode(f, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
}

// thumbPathFor returns the thumbnail_path currently stored for a file ID.
func thumbPathFor(t *testing.T, a *App, id int64) string {
	t.Helper()
	var p *string
	if err := a.db.Conn().QueryRow("SELECT thumbnail_path FROM files WHERE id = ?", id).Scan(&p); err != nil {
		t.Fatalf("query thumbnail_path: %v", err)
	}
	if p == nil {
		return ""
	}
	return *p
}

// TestRepairAllThumbnails verifies the vault-wide repair action (fix 2.5):
// files with an empty thumbnail_path get generated, and old-vault rows whose
// thumbnail_path points at a stale path get re-pointed to the canonical
// hashed path (with the WebP actually present on disk).
func TestRepairAllThumbnails(t *testing.T) {
	if !ffmpegAvailable {
		t.Skip("ffmpeg not available")
	}
	a := newTestApp(t)

	// img1.png — fresh file, no thumbnail yet.
	writeTestPNG(t, filepath.Join(a.vaultPath, "img1.png"))
	id1 := seedFile(t, a, "img1.png")

	// img2.png — old-vault row: thumbnail_path set but stale (no WebP there).
	writeTestPNG(t, filepath.Join(a.vaultPath, "img2.png"))
	id2 := seedFile(t, a, "img2.png")
	stalePath := ".tagloom/thumbnails/zz/0000deadbeef.webp"
	if _, err := a.db.Conn().Exec("UPDATE files SET thumbnail_path = ? WHERE id = ?", stalePath, id2); err != nil {
		t.Fatal(err)
	}

	// The normal pool must generate img1 but NOT touch img2 (its path is
	// non-empty, so it is invisible to the plain pool).
	if err := a.GenerateThumbnailsPool(); err != nil {
		t.Fatalf("GenerateThumbnailsPool: %v", err)
	}
	if got := thumbPathFor(t, a, id1); got == "" {
		t.Error("expected img1 thumbnail_path to be set by pool")
	}
	if !fileExists(t, filepath.Join(a.vaultPath, thumbPathFor(t, a, id1))) {
		t.Error("expected img1 WebP to exist on disk")
	}
	if got := thumbPathFor(t, a, id2); got != stalePath {
		t.Fatalf("plain pool must not touch stale rows, got %q", got)
	}

	// Repair fixes the stale row and leaves the good one alone.
	if err := a.RepairAllThumbnails(); err != nil {
		t.Fatalf("RepairAllThumbnails: %v", err)
	}
	wantRel := a.vault().toRelativePath(a.vault().generateThumbnailAbsolutePath("img2.png"))
	if got := thumbPathFor(t, a, id2); got != wantRel {
		t.Errorf("img2 thumbnail_path = %q, want canonical %q", got, wantRel)
	}
	if !fileExists(t, filepath.Join(a.vaultPath, wantRel)) {
		t.Error("expected img2 WebP to exist at canonical path")
	}
	if got := thumbPathFor(t, a, id1); got == "" {
		t.Error("img1 thumbnail_path should still be set")
	}

	// Repair is idempotent — second run must succeed and change nothing.
	before1 := thumbPathFor(t, a, id1)
	before2 := thumbPathFor(t, a, id2)
	if err := a.RepairAllThumbnails(); err != nil {
		t.Fatalf("second RepairAllThumbnails: %v", err)
	}
	if got := thumbPathFor(t, a, id1); got != before1 {
		t.Errorf("img1 path changed on second repair: %q", got)
	}
	if got := thumbPathFor(t, a, id2); got != before2 {
		t.Errorf("img2 path changed on second repair: %q", got)
	}
}

func fileExists(t *testing.T, path string) bool {
	t.Helper()
	_, err := os.Stat(path)
	return err == nil
}
