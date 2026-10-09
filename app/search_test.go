package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"TagLoom/db"
)

// TestSearchFilesEscaping verifies that special characters in the search
// query (SQL/FTS quoting characters, FTS operators, LIKE wildcards) are
// treated as literal text: no error, no injected clauses, no wildcard
// expansion.
func TestSearchFilesEscaping(t *testing.T) {
	a := newTestApp(t)
	seedFile(t, a, "sunset.jpg")

	// Queries that previously broke out of string literals (or would expand
	// as wildcards) must return zero rows and no error.
	for _, q := range []string{
		`o'brien`,                    // single quote in SQL string literal
		`say "hi"`,                   // double quote — old %q escaping produced \"
		`" OR 1=1 --`,                // SQL injection attempt
		`(' OR 1=1)`,                 // FTS paren operator + injection
		`sunset' OR 1=1 --`,          // would have matched everything if injected
		`_`,                          // LIKE wildcard must match literally
		`%`,                          // LIKE wildcard must match literally
		`* OR *`,                     // FTS OR operator must be literal
	} {
		files, err := a.SearchFiles(q, 50)
		if err != nil {
			t.Fatalf("SearchFiles(%q) error: %v", q, err)
		}
		if len(files) != 0 {
			t.Errorf("SearchFiles(%q) = %d files, want 0", q, len(files))
		}
	}

	// A legitimate query must still match via filename LIKE
	files, err := a.SearchFiles("sunset", 50)
	if err != nil {
		t.Fatalf("SearchFiles(sunset) error: %v", err)
	}
	if len(files) != 1 {
		t.Errorf("SearchFiles(sunset) = %d files, want 1", len(files))
	}

	// Whitespace-only query falls back to match-all FTS, but LIKE on "" still
	// matches everything — must not error.
	if _, err := a.SearchFiles("   ", 50); err != nil {
		t.Errorf("SearchFiles(whitespace) error: %v", err)
	}
}

// TestSearchFilesTagEscaping verifies the tag-matching branch (words ≥ 2
// chars trigger tag LIKE conditions) is also parameterized.
func TestSearchFilesTagEscaping(t *testing.T) {
	a := newTestApp(t)
	id := seedFile(t, a, "beach.jpg")
	tagID := seedTag(t, a, `o'brien red`)
	if _, err := a.db.Conn().Exec(
		"INSERT INTO file_tags (file_id, tag_id) VALUES (?, ?)", id, tagID); err != nil {
		t.Fatalf("insert file_tags: %v", err)
	}

	// Matching tag name (contains a quote) — must match via literal comparison
	files, err := a.SearchFiles(`o'brien`, 50)
	if err != nil {
		t.Fatalf("SearchFiles error: %v", err)
	}
	if len(files) != 1 {
		t.Errorf("SearchFiles(o'brien) = %d files, want 1 (tag match)", len(files))
	}

	// Injection-style tag word must not match or error
	files, err = a.SearchFiles(`x' OR 1=1 --`, 50)
	if err != nil {
		t.Fatalf("SearchFiles injection error: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("SearchFiles(injection) = %d files, want 0", len(files))
	}
}

// TestSanitizeFTSQuery verifies each word is returned as a double-quoted FTS5
// phrase with embedded quotes doubled — making every other character literal.
func TestSanitizeFTSQuery(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"plain", `"plain"`},
		{`he said "hi"`, `"he said ""hi"""`},
		{`a"b"c`, `"a""b""c"`},
		{"a*b?c", `"a*b?c"`},
		{"a:b (c) [d] {e} ~f &g |h -i", `"a:b (c) [d] {e} ~f &g |h -i"`},
		{`back\slash`, `"back\slash"`},
	}
	for _, tc := range tests {
		if got := sanitizeFTSQuery(tc.in); got != tc.want {
			t.Errorf("sanitizeFTSQuery(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestEscapeLike verifies LIKE wildcards and the escape character are escaped.
func TestEscapeLike(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"plain", "plain"},
		{"100%", "100\\%"},
		{"a_b", "a\\_b"},
		{`\`, `\\`},
		{`50%_x\y`, `50\%\_x\\y`},
	}
	for _, tc := range tests {
		if got := escapeLike(tc.in); got != tc.want {
			t.Errorf("escapeLike(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestSearchFilesFTS verifies the FTS branch (name/notes) still matches and
// that a quote in the query doesn't break the FTS match.
func TestSearchFilesFTS(t *testing.T) {
	a := newTestApp(t)
	// Insert with an explicit user name/notes so FTS has content
	if _, err := a.db.Conn().Exec(`
		INSERT INTO files (vault_path, folder_path, filename, name, notes,
		                   date_created, date_modified, indexed_at)
		VALUES ('portrait.jpg', '.', 'portrait.jpg', 'Alice Portrait',
		        'notes about the shoot', '2025-01-01T00:00:00Z',
		        '2025-01-01T00:00:00Z', '2025-01-01T00:00:00Z')`); err != nil {
		t.Fatalf("insert: %v", err)
	}

	files, err := a.SearchFiles("alice", 50)
	if err != nil {
		t.Fatalf("SearchFiles(alice) error: %v", err)
	}
	if len(files) != 1 {
		t.Errorf("SearchFiles(alice) = %d files, want 1", len(files))
	}

	// Quote inside the query must not error or match
	files, err = a.SearchFiles("alice's", 50)
	if err != nil {
		t.Fatalf("SearchFiles(alice's) error: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("SearchFiles(alice's) = %d files, want 0", len(files))
	}
}

// TestScanVaultStoresFileSize verifies ScanVault persists the file size
// (bytes) at index time so GetFiles can sort by file_size without
// filesystem stats.
func TestScanVaultStoresFileSize(t *testing.T) {
	a := newTestApp(t)

	content := strings.Repeat("x", 1234)
	if err := os.WriteFile(filepath.Join(a.vaultPath, "big.jpg"), []byte(content), 0600); err != nil {
		t.Fatalf("write: %v", err)
	}

	if _, err := a.ScanVault(); err != nil {
		t.Fatalf("ScanVault: %v", err)
	}

	var size int64
	if err := a.db.Conn().QueryRow(
		"SELECT file_size FROM files WHERE filename = 'big.jpg'").Scan(&size); err != nil {
		t.Fatalf("query file_size: %v", err)
	}
	if size != int64(len(content)) {
		t.Errorf("file_size = %d, want %d", size, len(content))
	}
}

// TestGetFilesSortByFileSize verifies file_size sorting is served from the
// DB (previously it stat-ed every matching file on every page request).
func TestGetFilesSortByFileSize(t *testing.T) {
	a := newTestApp(t)

	writeTestFile(t, a.vaultPath, "small.jpg") // 4 bytes ("test")
	if err := os.WriteFile(filepath.Join(a.vaultPath, "big.jpg"), []byte(strings.Repeat("y", 100)), 0600); err != nil {
		t.Fatalf("write: %v", err)
	}

	if _, err := a.ScanVault(); err != nil {
		t.Fatalf("ScanVault: %v", err)
	}

	page, err := a.GetFiles(db.FileFilter{}, db.SortOpts{Field: "file_size", Order: "asc"}, 0, 10)
	if err != nil {
		t.Fatalf("GetFiles: %v", err)
	}
	if len(page.Files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(page.Files))
	}
	if page.Files[0].Filename != "small.jpg" || page.Files[1].Filename != "big.jpg" {
		t.Errorf("asc order: %s then %s, want small.jpg then big.jpg",
			page.Files[0].Filename, page.Files[1].Filename)
	}

	page, err = a.GetFiles(db.FileFilter{}, db.SortOpts{Field: "file_size", Order: "desc"}, 0, 10)
	if err != nil {
		t.Fatalf("GetFiles desc: %v", err)
	}
	if page.Files[0].Filename != "big.jpg" {
		t.Errorf("desc order: first file is %s, want big.jpg", page.Files[0].Filename)
	}
}

// TestGetFilesSearchQueryPagination verifies that free-text search runs as a
// GetFiles filter: total_count counts every match (not just the page), pages
// continue past the page size, and the query ANDs with the other filters.
func TestGetFilesSearchQueryPagination(t *testing.T) {
	a := newTestApp(t)

	const total = 25
	var ids []int64
	for i := 0; i < total; i++ {
		ids = append(ids, seedFile(t, a, fmt.Sprintf("photos/sunset-%02d.jpg", i)))
	}
	for i := 0; i < 5; i++ {
		seedFile(t, a, fmt.Sprintf("photos/cat-%02d.jpg", i))
	}

	filter := db.FileFilter{SearchQuery: "sunset"}

	// Page 1 of 3 — total_count must be the full match count, not the page size.
	page, err := a.GetFiles(filter, db.SortOpts{Field: "filename", Order: "asc"}, 0, 10)
	if err != nil {
		t.Fatalf("GetFiles page 0: %v", err)
	}
	if page.TotalCount != total {
		t.Errorf("total_count = %d, want %d", page.TotalCount, total)
	}
	if len(page.Files) != 10 || page.Files[0].Filename != "sunset-00.jpg" {
		t.Fatalf("page 0 = %d files (first %q)", len(page.Files), page.Files[0].Filename)
	}

	// Pages keep going past the old 500-row search cap: page 2 holds the last 5.
	page, err = a.GetFiles(filter, db.SortOpts{Field: "filename", Order: "asc"}, 2, 10)
	if err != nil {
		t.Fatalf("GetFiles page 2: %v", err)
	}
	if len(page.Files) != 5 || page.Files[4].Filename != "sunset-24.jpg" {
		t.Fatalf("page 2 = %d files (last %q)", len(page.Files), page.Files[4].Filename)
	}

	// Search ANDs with the other filters, and GetFileIDs (batch edit scope)
	// sees the same set.
	tagID := seedTag(t, a, "golden")
	for _, id := range ids[:7] {
		if _, err := a.db.Conn().Exec(
			"INSERT INTO file_tags (file_id, tag_id) VALUES (?, ?)", id, tagID); err != nil {
			t.Fatalf("tag file: %v", err)
		}
	}
	filtered := db.FileFilter{SearchQuery: "sunset", TagGroups: [][]int64{{tagID}}}
	page, err = a.GetFiles(filtered, db.SortOpts{Field: "filename", Order: "asc"}, 0, 50)
	if err != nil {
		t.Fatalf("GetFiles tagged: %v", err)
	}
	if page.TotalCount != 7 || len(page.Files) != 7 {
		t.Errorf("tagged search = %d total / %d files, want 7/7", page.TotalCount, len(page.Files))
	}
	gotIDs, err := a.GetFileIDs(filtered)
	if err != nil {
		t.Fatalf("GetFileIDs: %v", err)
	}
	if len(gotIDs) != 7 {
		t.Errorf("GetFileIDs = %d ids, want 7", len(gotIDs))
	}
}

// TestSearchFilesNoVault verifies the guard.
func TestSearchFilesNoVault(t *testing.T) {
	a := &App{}
	_, err := a.SearchFiles("x", 10)
	if err == nil {
		t.Fatal("expected error with no vault open")
	}
	if !strings.Contains(err.Error(), "no vault") {
		t.Errorf("unexpected error: %v", err)
	}
}
