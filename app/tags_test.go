package app

import (
	"sort"
	"testing"

	"TagLoom/db"
)

// aliasSet returns a tag's aliases as a sorted set for order-independent compare.
func aliasSet(t *testing.T, a *App, tagID int64) []string {
	t.Helper()
	aliases, err := a.GetTagAliases(tagID)
	if err != nil {
		t.Fatalf("get aliases for tag %d: %v", tagID, err)
	}
	if aliases == nil {
		aliases = []string{}
	}
	sort.Strings(aliases)
	return aliases
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestTagAliasesAreArrays covers the alias payload change from a
// comma-separated string to []string: a comma inside an alias must survive as
// part of one alias (the old split turned "a,b" into two), whitespace is
// trimmed, and case-insensitive duplicates collapse.
func TestTagAliasesAreArrays(t *testing.T) {
	a := newTestApp(t)

	created, err := a.CreateTag(&db.TagCreate{
		Name:    "alice",
		Aliases: []string{"a,b", "  Alice  ", "A,B", "", "   "},
	})
	if err != nil {
		t.Fatalf("CreateTag: %v", err)
	}

	got := aliasSet(t, a, created.ID)
	want := []string{"Alice", "a,b"}
	if !equalStrings(got, want) {
		t.Errorf("aliases after create = %q, want %q", got, want)
	}

	// An empty list replaces the stored aliases — i.e. clears them.
	if err := a.UpdateTag(&db.TagUpdate{ID: created.ID, Name: "alice"}); err != nil {
		t.Fatalf("UpdateTag: %v", err)
	}
	if got := aliasSet(t, a, created.ID); len(got) != 0 {
		t.Errorf("aliases after clearing update = %q, want none", got)
	}

	// Re-submit two aliases, then confirm an alias owned by another tag is not
	// stolen: the case-insensitive unique index forbids it, and the write must
	// still succeed because aliases are best-effort.
	if err := a.UpdateTag(&db.TagUpdate{ID: created.ID, Name: "alice", Aliases: []string{"keep", "mine"}}); err != nil {
		t.Fatalf("UpdateTag: %v", err)
	}
	otherID := seedTag(t, a, "bob")
	if err := a.UpdateTag(&db.TagUpdate{ID: otherID, Name: "bob", Aliases: []string{"Mine", "theirs"}}); err != nil {
		t.Fatalf("UpdateTag (other tag): %v", err)
	}

	if got := aliasSet(t, a, created.ID); !equalStrings(got, []string{"keep", "mine"}) {
		t.Errorf("first tag aliases = %q, want [keep mine]", got)
	}
	if got := aliasSet(t, a, otherID); !equalStrings(got, []string{"theirs"}) {
		t.Errorf("other tag aliases = %q, want [theirs] — 'Mine' must not be stolen", got)
	}
}
