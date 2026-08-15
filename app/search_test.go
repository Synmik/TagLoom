package app

import (
	"strings"
	"testing"
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
