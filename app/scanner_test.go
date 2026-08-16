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

func TestResolveParent(t *testing.T) {
	vault := t.TempDir()

	tests := []struct {
		name  string
		fPath string
		want  string
	}{
		{"direct child of vault root", "photo.jpg", vault},
		{"dot path", ".", vault},
		{"one level deep", filepath.Join("photos", "vacation"), "photos"},
		{"multiple levels deep", filepath.Join("a", "b", "c"), filepath.Join("a", "b")},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := resolveParent(tc.fPath, vault); got != tc.want {
				t.Errorf("resolveParent(%q) = %q, want %q", tc.fPath, got, tc.want)
			}
		})
	}
}

func TestBuildFolderTree(t *testing.T) {
	root := t.TempDir()

	// Root has 2 files directly; "b" and "a" are subfolders (listed out of
	// order to verify alphabetical sorting); "a/inner" is an intermediate
	// folder with no direct files.
	counts := map[string]int{
		root:                        2,
		"a":                         1,
		filepath.Join("a", "inner"): 0,
		"b":                         3,
	}
	childrenOf := map[string][]string{
		root: {"b", "a"},
		"a":  {filepath.Join("a", "inner")},
	}

	node := buildFolderTree(counts, childrenOf, root, make(map[string]bool))

	if node.Path != root {
		t.Fatalf("root path = %q, want %q", node.Path, root)
	}
	if node.FileCount != 2 {
		t.Errorf("root file count = %d, want 2", node.FileCount)
	}
	if len(node.Children) != 2 {
		t.Fatalf("root children = %d, want 2", len(node.Children))
	}
	// Alphabetical order: "a" before "b"
	if node.Children[0].Name != "a" || node.Children[1].Name != "b" {
		t.Errorf("root children order = [%q, %q], want [a, b]",
			node.Children[0].Name, node.Children[1].Name)
	}
	if node.Children[0].FileCount != 1 {
		t.Errorf("a file count = %d, want 1", node.Children[0].FileCount)
	}
	if node.Children[1].FileCount != 3 {
		t.Errorf("b file count = %d, want 3", node.Children[1].FileCount)
	}
	// "a" contains the intermediate folder "inner" with count 0
	if len(node.Children[0].Children) != 1 {
		t.Fatalf("a children = %d, want 1", len(node.Children[0].Children))
	}
	if node.Children[0].Children[0].Name != "inner" ||
		node.Children[0].Children[0].FileCount != 0 {
		t.Errorf("inner child = %q (count %d), want inner (count 0)",
			node.Children[0].Children[0].Name, node.Children[0].Children[0].FileCount)
	}
}

// TestBuildFolderTreeSkipsCycles verifies the seen-map guard terminates and
// drops a folder that is listed as its own ancestor.
func TestBuildFolderTreeSkipsCycles(t *testing.T) {
	root := t.TempDir()

	counts := map[string]int{root: 0, "a": 1}
	childrenOf := map[string][]string{
		root: {"a"},
		"a":  {"a"}, // self-cycle
	}

	node := buildFolderTree(counts, childrenOf, root, make(map[string]bool))
	if len(node.Children) != 1 {
		t.Fatalf("root children = %d, want 1", len(node.Children))
	}
	if len(node.Children[0].Children) != 0 {
		t.Errorf("a children = %d, want 0 (cycle must be skipped)", len(node.Children[0].Children))
	}
}

// assertImported verifies an ImportResult reports exactly one successful import.
func assertImported(t *testing.T, r *ImportResult) {
	t.Helper()
	if r.Imported != 1 {
		t.Errorf("expected imported=1, got %+v", r)
	}
	if len(r.Errors) != 0 {
		t.Errorf("unexpected errors: %v", r.Errors)
	}
}

func TestImportFile(t *testing.T) {
	a := newTestApp(t)
	src := t.TempDir()
	photo := writeTestFile(t, src, "photo.jpg")

	// Copy import into vault root
	r := a.ImportFile(photo, false, "")
	assertImported(t, r)
	if _, err := os.Stat(filepath.Join(a.vaultPath, "photo.jpg")); err != nil {
		t.Errorf("imported file missing: %v", err)
	}
	if _, err := os.Stat(photo); err != nil {
		t.Errorf("source removed after copy import: %v", err)
	}

	// Duplicate name → "photo (2).jpg"
	r = a.ImportFile(photo, false, "")
	assertImported(t, r)
	if _, err := os.Stat(filepath.Join(a.vaultPath, "photo (2).jpg")); err != nil {
		t.Errorf("duplicate import did not create \"photo (2).jpg\": %v", err)
	}

	// Third import → "photo (3).jpg"
	r = a.ImportFile(photo, false, "")
	assertImported(t, r)
	if _, err := os.Stat(filepath.Join(a.vaultPath, "photo (3).jpg")); err != nil {
		t.Errorf("third import did not create \"photo (3).jpg\": %v", err)
	}

	// Move import into a nested target folder (created on the fly),
	// also hitting the duplicate branch in that folder
	clip := writeTestFile(t, src, "clip.mp4")
	r = a.ImportFile(clip, true, filepath.Join("sub", "dir"))
	assertImported(t, r)
	if _, err := os.Stat(filepath.Join(a.vaultPath, "sub", "dir", "clip.mp4")); err != nil {
		t.Errorf("moved file missing: %v", err)
	}
	if _, err := os.Stat(clip); !os.IsNotExist(err) {
		t.Errorf("source not removed after move import")
	}

	if got := countRows(t, a, "files"); got != 4 {
		t.Errorf("files table rows = %d, want 4", got)
	}
}

func TestImportFileSkipsAlreadyIndexed(t *testing.T) {
	a := newTestApp(t)
	src := t.TempDir()
	photo := writeTestFile(t, src, "photo.jpg")

	// Indexed in the DB but not on disk (e.g. deleted manually).
	// Importing a file with the same name hits the already-indexed branch.
	seedFile(t, a, "photo.jpg")

	r := a.ImportFile(photo, false, "")
	if r.Skipped != 1 {
		t.Errorf("expected skipped=1, got %+v", r)
	}
	if r.Imported != 0 {
		t.Errorf("expected imported=0, got %+v", r)
	}
	if got := countRows(t, a, "files"); got != 1 {
		t.Errorf("files table rows = %d, want 1", got)
	}
}

func TestImportFileErrors(t *testing.T) {
	a := newTestApp(t)
	src := t.TempDir()

	r := a.ImportFile(filepath.Join(src, "nope.jpg"), false, "")
	if r.Imported != 0 || len(r.Errors) == 0 {
		t.Errorf("missing source: expected error, got %+v", r)
	}

	r = a.ImportFile(src, false, "")
	if r.Imported != 0 || len(r.Errors) == 0 {
		t.Errorf("directory source: expected error, got %+v", r)
	}

	txt := writeTestFile(t, src, "notes.txt")
	r = a.ImportFile(txt, false, "")
	if r.Imported != 0 || len(r.Errors) == 0 {
		t.Errorf("unsupported type: expected error, got %+v", r)
	}

	if got := countRows(t, a, "files"); got != 0 {
		t.Errorf("files table rows = %d, want 0", got)
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
