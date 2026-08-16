package app

import (
	"os"
	"path/filepath"
	"testing"
)

// seedThumbnail creates a fake WebP under the canonical thumbnail tree and
// points the given file row at it. The name keeps paths unique within a
// test. Returns the absolute WebP path.
func seedThumbnail(t *testing.T, a *App, id int64, name string) string {
	t.Helper()
	thumbAbs := filepath.Join(a.vaultPath, ".tagloom", "thumbnails", "ab", name+".webp")
	if err := os.MkdirAll(filepath.Dir(thumbAbs), 0755); err != nil {
		t.Fatalf("mkdir thumb dir: %v", err)
	}
	if err := os.WriteFile(thumbAbs, []byte("webp"), 0600); err != nil {
		t.Fatalf("write thumb: %v", err)
	}
	rel := filepath.ToSlash(filepath.Join(".tagloom", "thumbnails", "ab", name+".webp"))
	if _, err := a.db.Conn().Exec("UPDATE files SET thumbnail_path = ? WHERE id = ?", rel, id); err != nil {
		t.Fatalf("set thumbnail_path: %v", err)
	}
	return thumbAbs
}

// TestDeleteFileRemovesThumbnail: deleting an indexed file must also remove
// its WebP (and the empty hash dir) — no orphan left behind.
func TestDeleteFileRemovesThumbnail(t *testing.T) {
	a := newTestApp(t)
	writeTestFile(t, a.vaultPath, "gone.png")
	id := seedFile(t, a, "gone.png")
	thumbAbs := seedThumbnail(t, a, id, "gone")

	if err := a.DeleteFile(id); err != nil {
		t.Fatalf("DeleteFile: %v", err)
	}
	if fileExists(t, thumbAbs) {
		t.Error("expected thumbnail to be removed after DeleteFile")
	}
	hashDir := filepath.Dir(thumbAbs)
	if fileExists(t, hashDir) {
		t.Error("expected empty hash directory to be removed")
	}
}

// TestDeleteFileKeepsThumbnailOfSurvivingFile: an untouched file keeps its
// thumbnail (guard against over-eager cleanup).
func TestDeleteFileKeepsThumbnailOfSurvivingFile(t *testing.T) {
	a := newTestApp(t)
	writeTestFile(t, a.vaultPath, "a.png")
	writeTestFile(t, a.vaultPath, "b.png")
	aID := seedFile(t, a, "a.png")
	bID := seedFile(t, a, "b.png")
	aThumb := seedThumbnail(t, a, aID, "aa")

	if err := a.DeleteFile(bID); err != nil {
		t.Fatalf("DeleteFile: %v", err)
	}
	if !fileExists(t, aThumb) {
		t.Error("surviving file's thumbnail must not be removed")
	}
}

// TestRescanRemovesOrphanThumbnails: when a rescan detects files that
// vanished from disk, their thumbnails must go too.
func TestRescanRemovesOrphanThumbnails(t *testing.T) {
	a := newTestApp(t)
	writeTestFile(t, a.vaultPath, "keep.jpg")
	writeTestFile(t, a.vaultPath, "removed.jpg")
	keepID := seedFile(t, a, "keep.jpg")
	removedID := seedFile(t, a, "removed.jpg")
	keepThumb := seedThumbnail(t, a, keepID, "keep")
	removedThumb := seedThumbnail(t, a, removedID, "removed")

	if err := os.Remove(filepath.Join(a.vaultPath, "removed.jpg")); err != nil {
		t.Fatalf("remove source: %v", err)
	}

	if _, err := a.RescanVault(); err != nil {
		t.Fatalf("RescanVault: %v", err)
	}

	if fileExists(t, removedThumb) {
		t.Error("expected thumbnail of removed file to be deleted by rescan")
	}
	if !fileExists(t, keepThumb) {
		t.Error("surviving file's thumbnail must not be removed by rescan")
	}
	var n int
	if err := a.db.Conn().QueryRow("SELECT COUNT(*) FROM files WHERE id = ?", keepID).Scan(&n); err != nil || n != 1 {
		t.Errorf("surviving file row should remain (n=%d, err=%v)", n, err)
	}
}
