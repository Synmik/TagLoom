package app

import (
	"path/filepath"
	"testing"

	"TagLoom/db"
)

func idSet(ids []int64) map[int64]bool {
	m := make(map[int64]bool, len(ids))
	for _, id := range ids {
		m[id] = true
	}
	return m
}

func expectIDs(t *testing.T, got []int64, want ...int64) {
	t.Helper()
	g := idSet(got)
	if len(g) != len(want) {
		t.Fatalf("got %d ids %v, want %v", len(g), got, want)
	}
	for _, w := range want {
		if !g[w] {
			t.Fatalf("ids %v missing %d", got, w)
		}
	}
}

// TestGetFileIDs verifies the ids-only query used by batch operations
// (fix 2.4): same filter semantics as GetFiles, bare IDs, no pagination.
func TestGetFileIDs(t *testing.T) {
	a := newTestApp(t)

	// a.jpg, b.png at vault root (folder "."), c.jpg inside "sub".
	aID := seedFile(t, a, "a.jpg")
	bID := seedFile(t, a, "b.png")
	cID := seedFile(t, a, filepath.Join("sub", "c.jpg"))

	// b.png: rating 5 + favorite. c.jpg: tagged.
	if _, err := a.db.Conn().Exec("UPDATE files SET rating = 5, is_favorite = 1 WHERE id = ?", bID); err != nil {
		t.Fatal(err)
	}
	tagID := seedTag(t, a, "landscape")
	if _, err := a.db.Conn().Exec("INSERT INTO file_tags (file_id, tag_id) VALUES (?, ?)", cID, tagID); err != nil {
		t.Fatal(err)
	}

	// No filter → all IDs.
	ids, err := a.GetFileIDs(db.FileFilter{})
	if err != nil {
		t.Fatal(err)
	}
	expectIDs(t, ids, aID, bID, cID)

	// Folder filter (relative) → only the file inside "sub".
	ids, err = a.GetFileIDs(db.FileFilter{FolderPath: "sub"})
	if err != nil {
		t.Fatal(err)
	}
	expectIDs(t, ids, cID)

	// Folder filter (absolute vault path → converted to ".") → root files.
	ids, err = a.GetFileIDs(db.FileFilter{FolderPath: a.vaultPath})
	if err != nil {
		t.Fatal(err)
	}
	expectIDs(t, ids, aID, bID)

	// Format filter → both jpgs.
	ids, err = a.GetFileIDs(db.FileFilter{FileFormats: []string{"jpg"}})
	if err != nil {
		t.Fatal(err)
	}
	expectIDs(t, ids, aID, cID)

	// Rating filter → only b.png.
	ids, err = a.GetFileIDs(db.FileFilter{MinRating: 4})
	if err != nil {
		t.Fatal(err)
	}
	expectIDs(t, ids, bID)

	// Favorites filter → only b.png.
	ids, err = a.GetFileIDs(db.FileFilter{FavoritesOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	expectIDs(t, ids, bID)

	// Untagged filter → a and b (c is tagged).
	ids, err = a.GetFileIDs(db.FileFilter{UntaggedOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	expectIDs(t, ids, aID, bID)

	// Combined: folder "sub" + favorites → empty (c is not a favorite).
	ids, err = a.GetFileIDs(db.FileFilter{FolderPath: "sub", FavoritesOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	if ids == nil {
		t.Fatal("expected non-nil empty slice for no matches")
	}
	if len(ids) != 0 {
		t.Fatalf("expected no ids, got %v", ids)
	}

	// Tag groups: files tagged with "landscape" → only c.
	ids, err = a.GetFileIDs(db.FileFilter{TagGroups: [][]int64{{tagID}}})
	if err != nil {
		t.Fatal(err)
	}
	expectIDs(t, ids, cID)

	// Sanity: GetFiles still agrees with GetFileIDs on the same filter.
	page, err := a.GetFiles(db.FileFilter{FileFormats: []string{"jpg"}}, db.SortOpts{Field: "indexed_at", Order: "desc"}, 0, 100)
	if err != nil {
		t.Fatal(err)
	}
	if page.TotalCount != 2 {
		t.Errorf("GetFiles TotalCount = %d, want 2", page.TotalCount)
	}
	jpgIDs, err := a.GetFileIDs(db.FileFilter{FileFormats: []string{"jpg"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(jpgIDs) != 2 {
		t.Errorf("GetFileIDs (jpg) count = %d, want 2", len(jpgIDs))
	}
}

// TestGetFileIDsNoVault verifies the binding-level guard.
func TestGetFileIDsNoVault(t *testing.T) {
	a := &App{}
	if _, err := a.GetFileIDs(db.FileFilter{}); err == nil {
		t.Fatal("expected error with no vault open")
	}
}

