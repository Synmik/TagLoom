package app

import (
	"os"
	"testing"
)

// TestFileCountCache verifies the cached files-table row count maintained by
// setFileCount / adjustFileCount / refreshFileCount (fix 2.3):
// GetCurrentVault must report accurate counts without re-querying COUNT(*).
func TestFileCountCache(t *testing.T) {
	a := newTestApp(t)

	// Fresh vault: 0, and GetCurrentVault serves it from the cache.
	info := a.GetCurrentVault()
	if info == nil || info.FileCount != 0 {
		t.Fatalf("new vault FileCount = %v, want 0", info)
	}

	// Two files indexed → +1 each via the indexFile path (adjustFileCount
	// when the row is new).
	p1 := writeTestFile(t, a.vaultPath, "a.jpg")
	p2 := writeTestFile(t, a.vaultPath, "b.png")
	if err := a.indexFile(p1); err != nil {
		t.Fatal(err)
	}
	if err := a.indexFile(p2); err != nil {
		t.Fatal(err)
	}
	if got := a.GetCurrentVault().FileCount; got != 2 {
		t.Errorf("after 2 indexFile: FileCount = %d, want 2", got)
	}

	// Re-indexing the same file is an upsert — count must not change.
	if err := a.indexFile(p1); err != nil {
		t.Fatal(err)
	}
	if got := a.GetCurrentVault().FileCount; got != 2 {
		t.Errorf("after re-index: FileCount = %d, want 2", got)
	}

	// ScanVault upserts and then refreshes from the DB.
	if _, err := a.ScanVault(); err != nil {
		t.Fatal(err)
	}
	if got := a.GetCurrentVault().FileCount; got != 2 {
		t.Errorf("after ScanVault: FileCount = %d, want 2", got)
	}

	// DeleteFile of an existing row → -1; of a missing row → unchanged.
	if err := a.DeleteFile(999999); err != nil {
		t.Fatal(err)
	}
	if got := a.GetCurrentVault().FileCount; got != 2 {
		t.Errorf("after deleting missing row: FileCount = %d, want 2", got)
	}
	// Seed a row directly (test helper bypasses the cache), sync the cache
	// the same way production would after external mutations, then delete.
	id := seedFile(t, a, "ghost.jpg")
	a.refreshFileCount(a.vault())
	if got := a.GetCurrentVault().FileCount; got != 3 {
		t.Fatalf("after seed + refresh: FileCount = %d, want 3", got)
	}
	if err := a.DeleteFile(id); err != nil {
		t.Fatal(err)
	}
	if got := a.GetCurrentVault().FileCount; got != 2 {
		t.Errorf("after DeleteFile: FileCount = %d, want 2", got)
	}

	// RescanVault: delete one file from disk → rescan removes it (delta -1).
	if err := os.Remove(p2); err != nil {
		t.Fatal(err)
	}
	if _, err := a.RescanVault(); err != nil {
		t.Fatal(err)
	}
	if got := a.GetCurrentVault().FileCount; got != 1 {
		t.Errorf("after RescanVault (-1): FileCount = %d, want 1", got)
	}

	// Add a new file on disk → rescan adds it (delta +1).
	writeTestFile(t, a.vaultPath, "c.gif")
	if _, err := a.RescanVault(); err != nil {
		t.Fatal(err)
	}
	if got := a.GetCurrentVault().FileCount; got != 2 {
		t.Errorf("after RescanVault (+1): FileCount = %d, want 2", got)
	}
}

// TestFileCountCacheConcurrent exercises the counter against the vault
// swap path (run under -race).
func TestFileCountCacheConcurrent(t *testing.T) {
	a := newTestApp(t)
	b := newTestApp(t)

	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			select {
			case <-stop:
				return
			default:
			}
			_ = a.GetCurrentVault()
			a.adjustFileCount(1)
			a.setFileCount(5)
			_ = a.vault()
		}
	}()

	for i := 0; i < 100; i++ {
		a.setVault(b.db, b.vaultPath, nil) // resets fileCount to 0
		a.setFileCount(i)
	}

	close(stop)
	<-done
}
