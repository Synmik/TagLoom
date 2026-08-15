package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"TagLoom/db"
)

// newTestApp creates an App backed by a real SQLite database in a temp vault
// directory. No Wails context is set, so emitEvent is a no-op.
func newTestApp(t *testing.T) *App {
	t.Helper()
	vault := t.TempDir()
	tagloomDir := filepath.Join(vault, ".tagloom")
	if err := os.MkdirAll(tagloomDir, 0755); err != nil {
		t.Fatalf("create .tagloom: %v", err)
	}
	d, err := db.NewDatabase(filepath.Join(tagloomDir, "tagloom.db"))
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { d.Close() })
	return &App{db: d, vaultPath: vault}
}

// writeTestFile creates a file with a supported extension on disk.
func writeTestFile(t *testing.T, dir, name string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	if err := os.WriteFile(p, []byte("test"), 0644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return p
}

// seedFile inserts a files row for a relative path and returns its ID.
func seedFile(t *testing.T, a *App, relPath string) int64 {
	t.Helper()
	res, err := a.db.Conn().Exec(`
		INSERT INTO files (vault_path, folder_path, filename, date_created, date_modified, indexed_at)
		VALUES (?, ?, ?, '2025-01-01T00:00:00Z', '2025-01-01T00:00:00Z', '2025-01-01T00:00:00Z')
	`, relPath, filepath.Dir(relPath), filepath.Base(relPath))
	if err != nil {
		t.Fatalf("insert files row: %v", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("last insert id: %v", err)
	}
	return id
}

// seedTag inserts a tag and returns its ID.
func seedTag(t *testing.T, a *App, name string) int64 {
	t.Helper()
	if _, err := a.db.Conn().Exec(
		"INSERT INTO tags (name, created_at) VALUES (?, '2025-01-01T00:00:00Z')", name); err != nil {
		t.Fatalf("insert tag: %v", err)
	}
	var id int64
	if err := a.db.Conn().QueryRow("SELECT id FROM tags WHERE name = ?", name).Scan(&id); err != nil {
		t.Fatalf("query tag id: %v", err)
	}
	return id
}

func countRows(t *testing.T, a *App, table string) int {
	t.Helper()
	var n int
	if err := a.db.Conn().QueryRow("SELECT COUNT(*) FROM " + table).Scan(&n); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return n
}

// TestRescanVaultRemovesOrphanedTags verifies the fix for the bug where
// RescanVault deleted from file_tags using a file *path* as the file_id,
// leaving orphaned tag associations behind.
func TestRescanVaultRemovesOrphanedTags(t *testing.T) {
	a := newTestApp(t)

	// Two files on disk, both indexed and both tagged
	writeTestFile(t, a.vaultPath, "keep.jpg")
	writeTestFile(t, a.vaultPath, "remove.png")
	keepID := seedFile(t, a, "keep.jpg")
	removeID := seedFile(t, a, "remove.png")

	tagID := seedTag(t, a, "red")
	for _, id := range []int64{keepID, removeID} {
		if _, err := a.db.Conn().Exec(
			"INSERT INTO file_tags (file_id, tag_id) VALUES (?, ?)", id, tagID); err != nil {
			t.Fatalf("insert file_tags: %v", err)
		}
	}

	if got := countRows(t, a, "files"); got != 2 {
		t.Fatalf("expected 2 files, got %d", got)
	}
	if got := countRows(t, a, "file_tags"); got != 2 {
		t.Fatalf("expected 2 file_tags, got %d", got)
	}

	// Remove one file from disk, then rescan
	if err := os.Remove(filepath.Join(a.vaultPath, "remove.png")); err != nil {
		t.Fatalf("remove file: %v", err)
	}

	added, err := a.RescanVault()
	if err != nil {
		t.Fatalf("RescanVault: %v", err)
	}
	if added != 0 {
		t.Errorf("expected 0 added files, got %d", added)
	}

	if got := countRows(t, a, "files"); got != 1 {
		t.Errorf("expected 1 file after rescan, got %d", got)
	}
	if got := countRows(t, a, "file_tags"); got != 1 {
		t.Errorf("expected 1 file_tags row (orphan not removed), got %d", got)
	}

	// The surviving file must keep its tag
	var keepTags int
	if err := a.db.Conn().QueryRow(
		"SELECT COUNT(*) FROM file_tags WHERE file_id = ?", keepID).Scan(&keepTags); err != nil {
		t.Fatalf("query keep tags: %v", err)
	}
	if keepTags != 1 {
		t.Errorf("expected 1 tag on surviving file, got %d", keepTags)
	}

	// The removed file must be fully gone
	var removeRows int
	if err := a.db.Conn().QueryRow(
		"SELECT COUNT(*) FROM files WHERE id = ?", removeID).Scan(&removeRows); err != nil {
		t.Fatalf("query removed file: %v", err)
	}
	if removeRows != 0 {
		t.Errorf("removed file still in files table")
	}
}

// TestRescanVaultAddsNewFiles verifies the rescan add path still works
// alongside the fixed delete path.
func TestRescanVaultAddsNewFiles(t *testing.T) {
	a := newTestApp(t)

	writeTestFile(t, a.vaultPath, "existing.jpg")
	seedFile(t, a, "existing.jpg")

	// New file appears on disk after initial index
	writeTestFile(t, a.vaultPath, "new.png")

	added, err := a.RescanVault()
	if err != nil {
		t.Fatalf("RescanVault: %v", err)
	}
	if added != 1 {
		t.Errorf("expected 1 added file, got %d", added)
	}
	if got := countRows(t, a, "files"); got != 2 {
		t.Errorf("expected 2 files after rescan, got %d", got)
	}
}

func TestIsExcluded(t *testing.T) {
	vault := t.TempDir()
	excluded := map[string]bool{
		strings.ToLower(filepath.Clean(filepath.Join(vault, "skip me"))): true,
	}

	tests := []struct {
		name string
		path string
		want bool
	}{
		{"tagloom dir itself", filepath.Join(vault, ".tagloom"), true},
		{"inside tagloom", filepath.Join(vault, ".tagloom", "thumbnails", "ab"), true},
		{"excluded folder", filepath.Join(vault, "skip me"), true},
		{"inside excluded folder", filepath.Join(vault, "skip me", "deep"), true},
		{"sibling of excluded", filepath.Join(vault, "skip me2"), false},
		{"normal folder", filepath.Join(vault, "photos"), false},
		{"vault root", vault, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isExcluded(tc.path, vault, excluded); got != tc.want {
				t.Errorf("isExcluded(%q) = %v, want %v", tc.path, got, tc.want)
			}
		})
	}
}

func TestResolveAndRelativePath(t *testing.T) {
	v := vault{path: t.TempDir()}

	rel := filepath.Join("photos", "vacation", "beach.jpg")
	sub := filepath.Join(v.path, "photos", "vacation", "beach.jpg")

	if got := v.toRelativePath(sub); got != rel {
		t.Errorf("toRelativePath = %q, want %q", got, rel)
	}
	if got := v.resolvePath(rel); got != sub {
		t.Errorf("resolvePath = %q, want %q", got, sub)
	}

	// Relative input passes through toRelativePath unchanged
	if got := v.toRelativePath(rel); got != rel {
		t.Errorf("toRelativePath(relative) = %q, want %q", got, rel)
	}
	// Absolute input outside vault passes through resolvePath unchanged (legacy)
	other := filepath.Join(t.TempDir(), "x.jpg")
	if got := v.resolvePath(other); got != other {
		t.Errorf("resolvePath(absolute) = %q, want %q", got, other)
	}
}
