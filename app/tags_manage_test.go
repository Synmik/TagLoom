package app

import (
	"database/sql"
	"strings"
	"testing"

	"TagLoom/db"
)

// mkTag creates a tag under parentID (nil = root) and returns its ID.
func mkTag(t *testing.T, a *App, name string, aliases []string, parentID *int64) int64 {
	t.Helper()
	tag, err := a.CreateTag(&db.TagCreate{Name: name, Aliases: aliases, ParentID: parentID})
	if err != nil {
		t.Fatalf("CreateTag %q: %v", name, err)
	}
	return tag.ID
}

// parentOf returns a tag's parent_id (nil when it is a root) and whether the tag exists.
func parentOf(t *testing.T, a *App, id int64) (*int64, bool) {
	t.Helper()
	var parent sql.NullInt64
	err := a.db.Conn().QueryRow("SELECT parent_id FROM tags WHERE id = ?", id).Scan(&parent)
	if err == sql.ErrNoRows {
		return nil, false
	}
	if err != nil {
		t.Fatalf("query parent of tag %d: %v", id, err)
	}
	if !parent.Valid {
		return nil, true
	}
	p := parent.Int64
	return &p, true
}

func tagExists(t *testing.T, a *App, id int64) bool {
	t.Helper()
	_, ok := parentOf(t, a, id)
	return ok
}

func linkCount(t *testing.T, a *App, tagID int64) int {
	t.Helper()
	var n int
	if err := a.db.Conn().QueryRow("SELECT COUNT(*) FROM file_tags WHERE tag_id = ?", tagID).Scan(&n); err != nil {
		t.Fatalf("count file_tags for tag %d: %v", tagID, err)
	}
	return n
}

func aliasCount(t *testing.T, a *App, tagID int64) int {
	t.Helper()
	var n int
	if err := a.db.Conn().QueryRow("SELECT COUNT(*) FROM tag_aliases WHERE tag_id = ?", tagID).Scan(&n); err != nil {
		t.Fatalf("count tag_aliases for tag %d: %v", tagID, err)
	}
	return n
}

// tree builds
//
//	root ─┬─ mid ─── leaf
//	    └─ sibling
//
// and returns their IDs plus three file IDs.
type tree struct {
	root, mid, leaf, sibling int64
	file1, file2, file3      int64
}

func buildTree(t *testing.T, a *App) tree {
	t.Helper()
	// Three files, unrelated to the tag tree until tagged below.
	for _, name := range []string{"one.jpg", "two.jpg", "three.jpg"} {
		writeTestFile(t, a.vaultPath, name)
	}
	tr := tree{
		file1: seedFile(t, a, "one.jpg"),
		file2: seedFile(t, a, "two.jpg"),
		file3: seedFile(t, a, "three.jpg"),
	}
	tr.root = mkTag(t, a, "root", []string{"r1", "r2"}, nil)
	tr.mid = mkTag(t, a, "mid", []string{"m1"}, &tr.root)
	tr.leaf = mkTag(t, a, "leaf", nil, &tr.mid)
	tr.sibling = mkTag(t, a, "sibling", nil, &tr.root)

	linkFile(t, a, tr.file1, tr.leaf)
	linkFile(t, a, tr.file2, tr.leaf)
	linkFile(t, a, tr.file3, tr.mid)
	return tr
}

func TestUpdateTagRejectsCycles(t *testing.T) {
	a := newTestApp(t)
	tr := buildTree(t, a)

	// Grandchild becoming the root's parent would close the loop; the root and
	// everything under it would stop being reachable from a root and vanish
	// from the tag tree.
	cases := []struct {
		name       string
		tagID      int64
		newParent  int64
		wantSubstr string
	}{
		{"self parent", tr.mid, tr.mid, "its own parent"},
		{"direct child", tr.root, tr.mid, "own descendant"},
		{"grandchild", tr.root, tr.leaf, "own descendant"},
	}
	for _, tc := range cases {
		err := a.UpdateTag(&db.TagUpdate{ID: tc.tagID, Name: nameOf(t, a, tc.tagID), ParentID: ptr(tc.newParent)})
		if err == nil {
			t.Errorf("%s: expected rejection, got nil error", tc.name)
			continue
		}
		if !strings.Contains(err.Error(), tc.wantSubstr) {
			t.Errorf("%s: error %q does not mention %q", tc.name, err.Error(), tc.wantSubstr)
		}
	}

	// A legitimate re-parent still works, and moving to the top level works.
	if err := a.UpdateTag(&db.TagUpdate{ID: tr.leaf, Name: "leaf", ParentID: ptr(tr.sibling)}); err != nil {
		t.Errorf("legitimate re-parent rejected: %v", err)
	}
	if got, _ := parentOf(t, a, tr.leaf); got == nil || *got != tr.sibling {
		t.Errorf("leaf parent = %v, want %d", got, tr.sibling)
	}
	if err := a.UpdateTag(&db.TagUpdate{ID: tr.leaf, Name: "leaf", ParentID: nil}); err != nil {
		t.Errorf("move to root rejected: %v", err)
	}
	if got, _ := parentOf(t, a, tr.leaf); got != nil {
		t.Errorf("leaf parent = %v, want nil", got)
	}
}

func TestUpdateTagRejectsMissingParent(t *testing.T) {
	a := newTestApp(t)
	tr := buildTree(t, a)

	err := a.UpdateTag(&db.TagUpdate{ID: tr.mid, Name: "mid", ParentID: ptr(int64(999999))})
	if err == nil || !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("expected missing-parent rejection, got %v", err)
	}
	if got, _ := parentOf(t, a, tr.mid); got == nil || *got != tr.root {
		t.Errorf("mid parent changed to %v after rejected update", got)
	}
}

func TestGetTagUsage(t *testing.T) {
	a := newTestApp(t)
	tr := buildTree(t, a)

	usage, err := a.GetTagUsage([]int64{tr.root, tr.mid, tr.leaf, tr.sibling, 999999})
	if err != nil {
		t.Fatalf("GetTagUsage: %v", err)
	}
	byID := map[int64]db.TagUsage{}
	for _, u := range usage {
		byID[u.TagID] = u
	}
	if len(usage) != 4 {
		t.Errorf("got %d usage rows, want 4 (unknown IDs must be skipped)", len(usage))
	}

	// root: 0 own files; subtree holds leaf's 2 + mid's 1 = 3; 2 children; 2 aliases
	root := byID[tr.root]
	if root.DirectFiles != 0 || root.DescendantFiles != 3 || root.Children != 2 || root.Aliases != 2 {
		t.Errorf("root usage = %+v, want direct 0 / descendants 3 / children 2 / aliases 2", root)
	}
	mid := byID[tr.mid]
	if mid.DirectFiles != 1 || mid.DescendantFiles != 3 || mid.Children != 1 || mid.Aliases != 1 {
		t.Errorf("mid usage = %+v, want direct 1 / descendants 3 / children 1 / aliases 1", mid)
	}
	leaf := byID[tr.leaf]
	if leaf.DirectFiles != 2 || leaf.DescendantFiles != 2 || leaf.Children != 0 || leaf.Aliases != 0 {
		t.Errorf("leaf usage = %+v, want direct 2 / descendants 2 / children 0 / aliases 0", leaf)
	}

	// Empty input is not an error, and duplicate IDs are collapsed.
	empty, err := a.GetTagUsage(nil)
	if err != nil || len(empty) != 0 {
		t.Errorf("GetTagUsage(nil) = %v, %v; want empty, nil", empty, err)
	}
	dup, err := a.GetTagUsage([]int64{tr.leaf, tr.leaf})
	if err != nil || len(dup) != 1 {
		t.Errorf("GetTagUsage with duplicate ids = %v, %v; want 1 row", dup, err)
	}
}

func TestGetAllTagAliases(t *testing.T) {
	a := newTestApp(t)
	tr := buildTree(t, a)

	aliases, err := a.GetAllTagAliases()
	if err != nil {
		t.Fatalf("GetAllTagAliases: %v", err)
	}
	got := map[int64][]string{}
	for _, al := range aliases {
		got[al.TagID] = append(got[al.TagID], al.Alias)
	}
	if strings.Join(got[tr.root], ",") != "r1,r2" {
		t.Errorf("root aliases = %v, want [r1 r2]", got[tr.root])
	}
	if strings.Join(got[tr.mid], ",") != "m1" {
		t.Errorf("mid aliases = %v, want [m1]", got[tr.mid])
	}
	if len(got[tr.leaf]) != 0 {
		t.Errorf("leaf aliases = %v, want none", got[tr.leaf])
	}
}

func TestDeleteTagsSafelyBlocksChildren(t *testing.T) {
	a := newTestApp(t)
	tr := buildTree(t, a)

	err := a.DeleteTagsSafely([]int64{tr.mid}, ChildStrategyBlock)
	if err == nil || !strings.Contains(err.Error(), "leaf") {
		t.Fatalf("expected block naming the child tag, got %v", err)
	}
	if !tagExists(t, a, tr.mid) || !tagExists(t, a, tr.leaf) {
		t.Error("block deleted tags; nothing should have been deleted")
	}
}

func TestDeleteTagsSafelyPromotesGrandchildren(t *testing.T) {
	a := newTestApp(t)
	tr := buildTree(t, a)

	if err := a.DeleteTagsSafely([]int64{tr.mid}, ChildStrategyPromote); err != nil {
		t.Fatalf("DeleteTagsSafely promote: %v", err)
	}
	if tagExists(t, a, tr.mid) {
		t.Fatal("mid should be deleted")
	}
	// The bug this replaces left leaf pointing at a deleted parent, which hid
	// it from the tag tree. It must now hang off root.
	if got, _ := parentOf(t, a, tr.leaf); got == nil || *got != tr.root {
		t.Errorf("leaf parent = %v, want root %d", got, tr.root)
	}
	if n := linkCount(t, a, tr.leaf); n != 2 {
		t.Errorf("leaf file links = %d, want 2 (untouched)", n)
	}
	if n := linkCount(t, a, tr.mid); n != 0 {
		t.Errorf("deleted tag still has %d file links", n)
	}
	if n := aliasCount(t, a, tr.mid); n != 0 {
		t.Errorf("deleted tag still has %d aliases", n)
	}
}

func TestDeleteTagsSafelyRootsChildren(t *testing.T) {
	a := newTestApp(t)
	tr := buildTree(t, a)

	if err := a.DeleteTagsSafely([]int64{tr.mid}, ChildStrategyRoot); err != nil {
		t.Fatalf("DeleteTagsSafely root: %v", err)
	}
	if got, _ := parentOf(t, a, tr.leaf); got != nil {
		t.Errorf("leaf parent = %v, want nil (top level)", got)
	}
}

func TestDeleteTagsSafelyCascade(t *testing.T) {
	a := newTestApp(t)
	tr := buildTree(t, a)

	if err := a.DeleteTagsSafely([]int64{tr.root}, ChildStrategyCascade); err != nil {
		t.Fatalf("DeleteTagsSafely cascade: %v", err)
	}
	for _, id := range []int64{tr.root, tr.mid, tr.leaf, tr.sibling} {
		if tagExists(t, a, id) {
			t.Errorf("tag %d should be cascade-deleted", id)
		}
	}
	if n := countRows(t, a, "file_tags"); n != 0 {
		t.Errorf("file_tags rows left = %d, want 0", n)
	}
	// Files themselves are never deleted by a tag operation.
	if n := countRows(t, a, "files"); n != 3 {
		t.Errorf("files left = %d, want 3", n)
	}
}

func TestDeleteTagsSafelyNestedDeleteNeverOrphans(t *testing.T) {
	a := newTestApp(t)
	tr := buildTree(t, a)

	// Deleting a tag and its child together: leaf's parent chain is entirely
	// inside the delete set, so it must land at the top level rather than keep
	// pointing at a deleted row.
	if err := a.DeleteTagsSafely([]int64{tr.root, tr.mid}, ChildStrategyPromote); err != nil {
		t.Fatalf("DeleteTagsSafely nested: %v", err)
	}
	if got, _ := parentOf(t, a, tr.leaf); got != nil {
		t.Errorf("leaf parent = %v, want nil", got)
	}
	if got, _ := parentOf(t, a, tr.sibling); got != nil {
		t.Errorf("sibling parent = %v, want nil", got)
	}
	var orphans int
	if err := a.db.Conn().QueryRow(
		`SELECT COUNT(*) FROM tags WHERE parent_id IS NOT NULL
		   AND parent_id NOT IN (SELECT id FROM tags)`,
	).Scan(&orphans); err != nil {
		t.Fatalf("count orphans: %v", err)
	}
	if orphans != 0 {
		t.Errorf("%d orphaned tags after nested delete", orphans)
	}
}

func TestDeleteTagsSafelyRejectsBadInput(t *testing.T) {
	a := newTestApp(t)
	tr := buildTree(t, a)

	if err := a.DeleteTagsSafely([]int64{tr.mid}, "sideways"); err == nil {
		t.Error("unknown strategy accepted; want error")
	}
	if err := a.DeleteTagsSafely([]int64{999999}, ChildStrategyPromote); err == nil {
		t.Error("unknown tag ID accepted; want error")
	}
	if err := a.DeleteTagsSafely(nil, ChildStrategyPromote); err != nil {
		t.Errorf("empty delete should be a no-op, got %v", err)
	}
	if !tagExists(t, a, tr.root) {
		t.Error("failed calls deleted tags")
	}
}

// linkFile associates a file with a tag.
func linkFile(t *testing.T, a *App, fileID, tagID int64) {
	t.Helper()
	if _, err := a.db.Conn().Exec("INSERT INTO file_tags (file_id, tag_id) VALUES (?, ?)", fileID, tagID); err != nil {
		t.Fatalf("link file %d to tag %d: %v", fileID, tagID, err)
	}
}

func ptr(v int64) *int64 { return &v }

func nameOf(t *testing.T, a *App, id int64) string {
	t.Helper()
	var name string
	if err := a.db.Conn().QueryRow("SELECT name FROM tags WHERE id = ?", id).Scan(&name); err != nil {
		t.Fatalf("query name of tag %d: %v", id, err)
	}
	return name
}

// twoFiles returns two indexed files.
func twoFiles(t *testing.T, a *App) (int64, int64) {
	t.Helper()
	for _, name := range []string{"one.jpg", "two.jpg"} {
		writeTestFile(t, a.vaultPath, name)
	}
	return seedFile(t, a, "one.jpg"), seedFile(t, a, "two.jpg")
}

func TestMergeTagsMovesFilesAndAliases(t *testing.T) {
	a := newTestApp(t)
	f1, f2 := twoFiles(t, a)

	src := mkTag(t, a, "misc", []string{"a1", "a2"}, nil)
	dst := mkTag(t, a, "keep", []string{"d1"}, nil)
	linkFile(t, a, f1, src)
	linkFile(t, a, f2, src)
	linkFile(t, a, f1, dst) // already carries the target — must be deduplicated

	res, err := a.MergeTags([]int64{src}, dst)
	if err != nil {
		t.Fatalf("MergeTags: %v", err)
	}
	// The UI reports these numbers instead of guessing them client-side.
	if res.Sources != 1 || res.FilesMoved != 1 || res.Duplicates != 1 {
		t.Errorf("merge result = %+v, want sources 1 / moved 1 / duplicates 1", res)
	}
	if res.AliasesMoved != 2 || res.NamesAdopted != 1 || res.AliasesSkipped != 0 {
		t.Errorf("merge aliases = %+v, want moved 2 / names adopted 1 / skipped 0", res)
	}

	if tagExists(t, a, src) {
		t.Error("source tag still exists after merge")
	}
	if n := linkCount(t, a, dst); n != 2 {
		t.Errorf("target file links = %d, want 2 (one moved, one deduplicated)", n)
	}
	if n := linkCount(t, a, src); n != 0 {
		t.Errorf("source still has %d file links", n)
	}
	if n := countRows(t, a, "files"); n != 2 {
		t.Errorf("files left = %d, want 2 — a merge never deletes files", n)
	}

	// The source's aliases move, and its name becomes an alias too, so a search
	// for the tag that no longer exists still resolves.
	got := aliasSet(t, a, dst)
	want := []string{"a1", "a2", "d1", "misc"}
	if !equalStrings(got, want) {
		t.Errorf("target aliases after merge = %q, want %q — aliases must always survive", got, want)
	}
	if n := aliasCount(t, a, src); n != 0 {
		t.Errorf("deleted source still has %d aliases", n)
	}
}

func TestMergeTagsMovesChildrenUnderTarget(t *testing.T) {
	a := newTestApp(t)
	tr := buildTree(t, a) // root ─┬─ mid ─── leaf, └─ sibling
	dst := mkTag(t, a, "dst", nil, nil)

	res, err := a.MergeTags([]int64{tr.mid}, dst)
	if err != nil {
		t.Fatalf("MergeTags: %v", err)
	}
	if res.ChildrenMoved != 1 {
		t.Errorf("children moved = %d, want 1", res.ChildrenMoved)
	}
	// leaf must not follow mid into deletion, nor keep pointing at it. A merge
	// replaces the source everywhere, so the subtree moves under the target.
	if got, _ := parentOf(t, a, tr.leaf); got == nil || *got != dst {
		t.Errorf("leaf parent = %v, want target %d", got, dst)
	}
	var orphans int
	if err := a.db.Conn().QueryRow(
		`SELECT COUNT(*) FROM tags WHERE parent_id IS NOT NULL
		   AND parent_id NOT IN (SELECT id FROM tags)`,
	).Scan(&orphans); err != nil {
		t.Fatalf("count orphans: %v", err)
	}
	if orphans != 0 {
		t.Errorf("%d orphaned tags after merge", orphans)
	}
}

func TestMergeTagsRejectsBadInput(t *testing.T) {
	a := newTestApp(t)
	tr := buildTree(t, a)
	category := mkTag(t, a, "Photo", nil, nil)
	if err := a.UpdateTag(&db.TagUpdate{ID: category, Name: "Photo", IsCategory: 1}); err != nil {
		t.Fatalf("make category: %v", err)
	}

	cases := []struct {
		name    string
		sources []int64
		target  int64
		want    string
	}{
		{"into itself", []int64{tr.leaf}, tr.leaf, "into itself"},
		{"missing source", []int64{999999}, tr.leaf, "does not exist"},
		{"missing target", []int64{tr.leaf}, 999999, "does not exist"},
		{"tag into category", []int64{tr.leaf}, category, `cannot merge tag "leaf" into category`},
		{"category into tag", []int64{category}, tr.leaf, `cannot merge category "Photo" into tag`},
		{"into own descendant", []int64{tr.root}, tr.leaf, "own descendant"},
		{"no sources", nil, tr.leaf, "at least one source"},
	}
	for _, tc := range cases {
		_, err := a.MergeTags(tc.sources, tc.target)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: got %v, want error mentioning %q", tc.name, err, tc.want)
		}
	}

	// None of the refusals may have deleted anything.
	for _, id := range []int64{tr.root, tr.mid, tr.leaf, tr.sibling, category} {
		if !tagExists(t, a, id) {
			t.Errorf("tag %d deleted by a rejected merge", id)
		}
	}
}

func TestMergeTagsMergesSeveralSourcesAtOnce(t *testing.T) {
	a := newTestApp(t)
	f1, f2 := twoFiles(t, a)

	s1 := mkTag(t, a, "one", []string{"x"}, nil)
	s2 := mkTag(t, a, "two", []string{"y"}, nil)
	dst := mkTag(t, a, "dst", nil, nil)
	linkFile(t, a, f1, s1)
	linkFile(t, a, f2, s2)

	if _, err := a.MergeTags([]int64{s1, s2}, dst); err != nil {
		t.Fatalf("MergeTags: %v", err)
	}
	if n := linkCount(t, a, dst); n != 2 {
		t.Errorf("target file links = %d, want 2", n)
	}
	if got := aliasSet(t, a, dst); !equalStrings(got, []string{"one", "two", "x", "y"}) {
		t.Errorf("target aliases = %q, want [one two x y]", got)
	}
	for _, id := range []int64{s1, s2} {
		if tagExists(t, a, id) {
			t.Errorf("source tag %d still exists", id)
		}
	}
}

func TestMergeTagsAdoptSourceNameKeepsSearchWorking(t *testing.T) {
	a := newTestApp(t)
	f1, _ := twoFiles(t, a)

	person := mkTag(t, a, "person", nil, nil)
	alice := mkTag(t, a, "alice", nil, nil)
	linkFile(t, a, f1, person)

	if _, err := a.MergeTags([]int64{person}, alice); err != nil {
		t.Fatalf("MergeTags: %v", err)
	}
	// This is what keeps a saved search for "person" resolving to the file after
	// the tag itself is gone — app/search.go joins tag_aliases.
	aliases, err := a.GetTagAliases(alice)
	if err != nil {
		t.Fatalf("GetTagAliases: %v", err)
	}
	if !equalStrings(aliases, []string{"person"}) {
		t.Errorf("target aliases = %q, want [person]", aliases)
	}

	hits, err := a.SearchFiles("person", 50)
	if err != nil {
		t.Fatalf("SearchFiles: %v", err)
	}
	if len(hits) != 1 {
		t.Errorf("search for the merged-away tag name found %d files, want 1", len(hits))
	}
}

func TestMergeCategoryIntoCategory(t *testing.T) {
	a := newTestApp(t)
	aPhoto := mkTag(t, a, "aPhoto", nil, nil)
	bPhoto := mkTag(t, a, "bPhoto", nil, nil)
	sub := mkTag(t, a, "Landscape", nil, &aPhoto)
	for _, id := range []int64{aPhoto, bPhoto} {
		if err := a.UpdateTag(&db.TagUpdate{ID: id, Name: nameOf(t, a, id), IsCategory: 1}); err != nil {
			t.Fatalf("make category: %v", err)
		}
	}

	if _, err := a.MergeTags([]int64{aPhoto}, bPhoto); err != nil {
		t.Fatalf("category into category: %v", err)
	}
	if tagExists(t, a, aPhoto) {
		t.Error("source category still exists")
	}
	if got, _ := parentOf(t, a, sub); got == nil || *got != bPhoto {
		t.Errorf("sub-tag parent = %v, want target category %d", got, bPhoto)
	}
}

func TestMergeTagsAdoptsColourOfUncolouredTarget(t *testing.T) {
	a := newTestApp(t)
	src, err := a.CreateTag(&db.TagCreate{Name: "coloured", Color: "#ff0000"})
	if err != nil {
		t.Fatalf("CreateTag: %v", err)
	}
	dst := mkTag(t, a, "plain", nil, nil) // no colour

	if _, err := a.MergeTags([]int64{src.ID}, dst); err != nil {
		t.Fatalf("MergeTags: %v", err)
	}
	var color string
	if err := a.db.Conn().QueryRow("SELECT color FROM tags WHERE id = ?", dst).Scan(&color); err != nil {
		t.Fatalf("query colour: %v", err)
	}
	if color != "#ff0000" {
		t.Errorf("target colour = %q, want #ff0000", color)
	}
}

func TestMoveTagRules(t *testing.T) {
	a := newTestApp(t)
	tr := buildTree(t, a)
	category := mkTag(t, a, "Photo", nil, nil)
	if err := a.UpdateTag(&db.TagUpdate{ID: category, Name: "Photo", IsCategory: 1}); err != nil {
		t.Fatalf("make category: %v", err)
	}

	// Rejections, while the tree is still in its original shape: root ─┬─ mid
	// ─── leaf, └─ sibling. root under leaf would close the loop through mid.
	if err := a.MoveTag(tr.root, ptr(tr.leaf), 0); err == nil || !strings.Contains(err.Error(), "descendant") {
		t.Errorf("cyclic move accepted (%v); want rejection naming the descendant", err)
	}
	if err := a.MoveTag(category, ptr(tr.root), 0); err == nil {
		t.Error("category given a parent; want rejection")
	}
	if err := a.MoveTag(tr.leaf, ptr(int64(999999)), 0); err == nil {
		t.Error("move under missing parent accepted; want rejection")
	}
	if got, _ := parentOf(t, a, tr.root); got != nil {
		t.Errorf("root parent = %v after rejected move, want nil", got)
	}
	if got, _ := parentOf(t, a, tr.leaf); got == nil || *got != tr.mid {
		t.Errorf("leaf parent = %v after rejected moves, want mid %d", got, tr.mid)
	}

	// A category groups tags, so it may be a parent.
	if err := a.MoveTag(tr.leaf, ptr(category), 0); err != nil {
		t.Fatalf("move under category: %v", err)
	}
	if got, _ := parentOf(t, a, tr.leaf); got == nil || *got != category {
		t.Errorf("leaf parent = %v, want category %d", got, category)
	}
	// And back to the top level.
	if err := a.MoveTag(tr.leaf, nil, 0); err != nil {
		t.Fatalf("move to top level: %v", err)
	}
	if got, _ := parentOf(t, a, tr.leaf); got != nil {
		t.Errorf("leaf parent = %v, want nil", got)
	}
}

func TestUpdateTagNormalizesCategoryParent(t *testing.T) {
	a := newTestApp(t)
	tr := buildTree(t, a)

	// Turning a nested tag into a category clears its parent rather than
	// rejecting the write, so legacy rows stay editable.
	if err := a.UpdateTag(&db.TagUpdate{ID: tr.mid, Name: "mid", IsCategory: 1, ParentID: ptr(tr.root)}); err != nil {
		t.Fatalf("UpdateTag: %v", err)
	}
	if got, _ := parentOf(t, a, tr.mid); got != nil {
		t.Errorf("category parent = %v, want nil", got)
	}
	// Its children are untouched — a category is a grouping tag.
	if got, _ := parentOf(t, a, tr.leaf); got == nil || *got != tr.mid {
		t.Errorf("leaf parent = %v, want mid %d", got, tr.mid)
	}
	// A category can be turned back into a normal tag and re-parented.
	if err := a.UpdateTag(&db.TagUpdate{ID: tr.mid, Name: "mid", IsCategory: 0, ParentID: ptr(tr.sibling)}); err != nil {
		t.Fatalf("UpdateTag (uncategorise): %v", err)
	}
	if got, _ := parentOf(t, a, tr.mid); got == nil || *got != tr.sibling {
		t.Errorf("mid parent = %v, want sibling %d", got, tr.sibling)
	}
}

// orderedNames lists a level of the hierarchy in the order GetTags returns it.
func orderedNames(t *testing.T, a *App, parentID *int64) []string {
	t.Helper()
	const orderBy = ` ORDER BY sort_order ASC, name ASC`
	var (
		rows *sql.Rows
		err  error
	)
	if parentID == nil {
		rows, err = a.db.Conn().Query(`SELECT name FROM tags WHERE parent_id IS NULL` + orderBy)
	} else {
		rows, err = a.db.Conn().Query(`SELECT name FROM tags WHERE parent_id = ?`+orderBy, *parentID)
	}
	if err != nil {
		t.Fatalf("query level: %v", err)
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan name: %v", err)
		}
		names = append(names, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("read level: %v", err)
	}
	return names
}

func TestMoveTagOrdersWithinLevel(t *testing.T) {
	a := newTestApp(t)
	// Four root tags, created in a deliberately scrambled order so name order
	// alone is not the starting order.
	// Create them in alphabetical order, then impose a hand-made order — the
	// state a level is in after the user has dragged rows around.
	for _, n := range []string{"delta", "alpha", "charlie", "bravo"} {
		if _, err := a.CreateTag(&db.TagCreate{Name: n}); err != nil {
			t.Fatalf("CreateTag %s: %v", n, err)
		}
	}
	for i, n := range []string{"delta", "alpha", "charlie", "bravo"} {
		if _, err := a.db.Conn().Exec("UPDATE tags SET sort_order = ? WHERE name = ?", i, n); err != nil {
			t.Fatalf("impose sort_order for %s: %v", n, err)
		}
	}
	if got := orderedNames(t, a, nil); !equalStrings(got, []string{"delta", "alpha", "charlie", "bravo"}) {
		t.Fatalf("initial root order = %q", got)
	}

	// Move bravo in front of alpha.
	bravoID := idOf(t, a, "bravo")
	alphaID := idOf(t, a, "alpha")
	if err := a.MoveTag(bravoID, nil, alphaID); err != nil {
		t.Fatalf("MoveTag before alpha: %v", err)
	}
	if got := orderedNames(t, a, nil); !equalStrings(got, []string{"delta", "bravo", "alpha", "charlie"}) {
		t.Errorf("order after move-before = %q, want [delta bravo alpha charlie]", got)
	}

	// beforeID = 0 appends to the end of the level.
	if err := a.MoveTag(bravoID, nil, 0); err != nil {
		t.Fatalf("MoveTag append: %v", err)
	}
	if got := orderedNames(t, a, nil); !equalStrings(got, []string{"delta", "alpha", "charlie", "bravo"}) {
		t.Errorf("order after append = %q, want [delta alpha charlie bravo]", got)
	}

	// A beforeID that is not a sibling of the destination is treated as append.
	other := mkTag(t, a, "nested", nil, ptr(idOf(t, a, "delta")))
	if err := a.MoveTag(bravoID, nil, other); err != nil {
		t.Fatalf("MoveTag with foreign beforeID: %v", err)
	}
	if got := orderedNames(t, a, nil); len(got) == 0 || got[len(got)-1] != "bravo" {
		t.Errorf("order after foreign-beforeID move = %q, want bravo last", got)
	}
}

func idOf(t *testing.T, a *App, name string) int64 {
	t.Helper()
	var id int64
	if err := a.db.Conn().QueryRow("SELECT id FROM tags WHERE name = ?", name).Scan(&id); err != nil {
		t.Fatalf("id of %s: %v", name, err)
	}
	return id
}
