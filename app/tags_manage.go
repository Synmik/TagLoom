package app

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"TagLoom/db"
	"TagLoom/utils"
)

// This file holds the tag-management bindings behind the Tag Manager modal:
// hierarchy validation, usage reporting, safe deletion, merging and moving.
// Tag CRUD itself lives in tags.go.
//
// Nothing here relies on SQLite cascades: PRAGMA foreign_keys is off (see
// db/schema.sql), so every parent/child and file-association consequence is
// handled with explicit SQL.

// Child-tag strategies accepted by DeleteTagsSafely.
const (
	// ChildStrategyPromote moves each child onto the nearest surviving
	// ancestor (its grandparent, normally).
	ChildStrategyPromote = "promote"
	// ChildStrategyRoot moves each surviving child to the top level.
	ChildStrategyRoot = "root"
	// ChildStrategyCascade deletes the whole subtree.
	ChildStrategyCascade = "cascade"
	// ChildStrategyBlock refuses to delete a tag that still has children.
	ChildStrategyBlock = "block"
)

// queryer is satisfied by both *sql.DB and *sql.Tx, so the read helpers below
// can run either outside a transaction or inside one.
type queryer interface {
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
}

// tagGraph is the whole tag hierarchy in memory. The tag table is small enough
// that loading it once is cheaper — and clearer — than a recursive query per
// validation.
type tagGraph struct {
	parent     map[int64]*int64
	children   map[int64][]int64
	names      map[int64]string
	colors     map[int64]string
	isCategory map[int64]bool
}

func loadTagGraph(q queryer) (*tagGraph, error) {
	rows, err := q.Query(`SELECT id, parent_id, name, color, is_category FROM tags`)
	if err != nil {
		return nil, fmt.Errorf("load tag graph: %w", err)
	}
	defer rows.Close()

	g := &tagGraph{
		parent:     map[int64]*int64{},
		children:   map[int64][]int64{},
		names:      map[int64]string{},
		colors:     map[int64]string{},
		isCategory: map[int64]bool{},
	}
	for rows.Next() {
		var (
			id       int64
			parentID *int64
			name     string
			color    sql.NullString // tags.color is nullable
			isCat    int
		)
		if err := rows.Scan(&id, &parentID, &name, &color, &isCat); err != nil {
			return nil, fmt.Errorf("scan tag graph: %w", err)
		}
		g.parent[id] = parentID
		g.names[id] = name
		g.colors[id] = color.String
		g.isCategory[id] = isCat == 1
		if parentID != nil {
			g.children[*parentID] = append(g.children[*parentID], id)
		}
	}
	return g, rows.Err()
}

// descendants returns every tag below id, in breadth-first order. Each tag is
// visited at most once, so stored data that already contains a cycle cannot
// loop forever here.
func (g *tagGraph) descendants(id int64) []int64 {
	var out []int64
	seen := map[int64]struct{}{id: {}}
	queue := append([]int64(nil), g.children[id]...)
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if _, dup := seen[cur]; dup {
			continue
		}
		seen[cur] = struct{}{}
		out = append(out, cur)
		queue = append(queue, g.children[cur]...)
	}
	return out
}

// isDescendant reports whether candidate sits somewhere below id.
func (g *tagGraph) isDescendant(id, candidate int64) bool {
	for _, d := range g.descendants(id) {
		if d == candidate {
			return true
		}
	}
	return false
}

// exists reports whether the tag is present in the vault.
func (g *tagGraph) exists(id int64) bool {
	_, ok := g.parent[id]
	return ok
}

// validateNewParent rejects a parent that would break the hierarchy: the tag
// itself, a tag that does not exist, or one of the tag's own descendants.
// A descendant parent is the silent-data-loss case — the subtree stops being
// reachable from a root, so it vanishes from the tag tree without any error.
func (a *App) validateNewParent(q queryer, tagID int64, newParentID *int64) error {
	if newParentID == nil {
		return nil
	}
	g, err := loadTagGraph(q)
	if err != nil {
		return err
	}
	return validateNewParentInGraph(g, tagID, newParentID)
}

// validateNewParentInGraph is validateNewParent against an already loaded
// graph, so a caller that holds a transaction does not load the hierarchy twice.
func validateNewParentInGraph(g *tagGraph, tagID int64, newParentID *int64) error {
	if newParentID == nil {
		return nil
	}
	if _, ok := g.parent[tagID]; !ok {
		return fmt.Errorf("tag %d does not exist", tagID)
	}
	if *newParentID == tagID {
		return fmt.Errorf("tag %q cannot be its own parent", g.names[tagID])
	}
	if !g.exists(*newParentID) {
		return fmt.Errorf("parent tag %d does not exist", *newParentID)
	}
	if g.isDescendant(tagID, *newParentID) {
		return fmt.Errorf("cannot move tag %q under its own descendant %q",
			g.names[tagID], g.names[*newParentID])
	}
	return nil
}

// GetTagUsage returns usage information for the given tags, for the delete and
// merge previews. IDs that do not exist are omitted from the result. An empty
// input yields an empty slice.
func (a *App) GetTagUsage(tagIDs []int64) ([]db.TagUsage, error) {
	v := a.vault()
	if v.db == nil {
		return nil, fmt.Errorf("no vault open")
	}
	if len(tagIDs) == 0 {
		return []db.TagUsage{}, nil
	}

	g, err := loadTagGraph(v.db.Conn())
	if err != nil {
		return nil, err
	}

	// Direct file counts for every tag in one query.
	direct := map[int64]int{}
	rows, err := v.db.Conn().Query(`SELECT tag_id, COUNT(*) FROM file_tags GROUP BY tag_id`)
	if err != nil {
		return nil, fmt.Errorf("count tag usages: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var tagID int64
		var count int
		if err := rows.Scan(&tagID, &count); err != nil {
			return nil, fmt.Errorf("scan tag usage: %w", err)
		}
		direct[tagID] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("scan tag usages: %w", err)
	}

	aliasCounts := map[int64]int{}
	aliasRows, err := v.db.Conn().Query(`SELECT tag_id, COUNT(*) FROM tag_aliases GROUP BY tag_id`)
	if err != nil {
		return nil, fmt.Errorf("count tag aliases: %w", err)
	}
	defer aliasRows.Close()
	for aliasRows.Next() {
		var tagID int64
		var count int
		if err := aliasRows.Scan(&tagID, &count); err != nil {
			return nil, fmt.Errorf("scan tag aliases: %w", err)
		}
		aliasCounts[tagID] = count
	}
	if err := aliasRows.Err(); err != nil {
		return nil, fmt.Errorf("scan tag alias counts: %w", err)
	}

	seen := map[int64]struct{}{}
	usage := make([]db.TagUsage, 0, len(tagIDs))
	for _, id := range tagIDs {
		if _, dup := seen[id]; dup {
			continue
		}
		if !g.exists(id) {
			continue
		}
		seen[id] = struct{}{}

		total := direct[id]
		for _, d := range g.descendants(id) {
			total += direct[d]
		}
		usage = append(usage, db.TagUsage{
			TagID:           id,
			Name:            g.names[id],
			IsCategory:      boolToInt(g.isCategory[id]),
			DirectFiles:     direct[id],
			DescendantFiles: total,
			Children:        len(g.children[id]),
			Aliases:         aliasCounts[id],
		})
	}
	return usage, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// GetAllTagAliases returns every alias with its owning tag, for the Tag
// Manager's alias-aware search and merge previews — one query instead of one
// per tag.
func (a *App) GetAllTagAliases() ([]db.TagAlias, error) {
	v := a.vault()
	if v.db == nil {
		return nil, fmt.Errorf("no vault open")
	}

	rows, err := v.db.Conn().Query(`SELECT tag_id, alias FROM tag_aliases ORDER BY alias ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	aliases := []db.TagAlias{}
	for rows.Next() {
		var ta db.TagAlias
		if err := rows.Scan(&ta.TagID, &ta.Alias); err != nil {
			return nil, err
		}
		aliases = append(aliases, ta)
	}
	return aliases, rows.Err()
}

// DeleteTagsSafely deletes tags in a single transaction without ever leaving a
// child tag pointing at a deleted parent — the failure mode of the original
// DeleteTag, which deleted the row and silently orphaned its subtree (foreign
// keys are not enforced, and the tag tree renders neither roots nor children of
// a missing parent, so those tags simply disappeared).
//
// childStrategy decides what happens to the children of a deleted tag:
// ChildStrategyPromote, ChildStrategyRoot, ChildStrategyCascade, or
// ChildStrategyBlock (refuse when children exist). Files are always
// un-tagged; files themselves are never touched.
func (a *App) DeleteTagsSafely(tagIDs []int64, childStrategy string) error {
	v := a.vault()
	if v.db == nil {
		return fmt.Errorf("no vault open")
	}
	if len(tagIDs) == 0 {
		return nil
	}
	switch childStrategy {
	case ChildStrategyPromote, ChildStrategyRoot, ChildStrategyCascade, ChildStrategyBlock:
	default:
		return fmt.Errorf("unknown child strategy %q (want promote, root, cascade or block)", childStrategy)
	}

	tx, err := v.db.Conn().Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	g, err := loadTagGraph(tx)
	if err != nil {
		return err
	}

	// Validate the inputs before touching anything.
	inputs := uniqueIDs(tagIDs)
	for _, id := range inputs {
		if !g.exists(id) {
			return fmt.Errorf("tag %d does not exist", id)
		}
	}

	// Work out the full set of rows to remove.
	deleteSet := map[int64]struct{}{}
	for _, id := range inputs {
		deleteSet[id] = struct{}{}
		if childStrategy == ChildStrategyCascade {
			for _, d := range g.descendants(id) {
				deleteSet[d] = struct{}{}
			}
		}
	}

	if childStrategy == ChildStrategyBlock {
		for _, id := range inputs {
			var kept []string
			for _, child := range g.children[id] {
				if _, deleting := deleteSet[child]; !deleting {
					kept = append(kept, g.names[child])
				}
			}
			if len(kept) > 0 {
				return fmt.Errorf("tag %q has %d child tag(s) (%s); choose what happens to them first",
					g.names[id], len(kept), previewNames(kept))
			}
		}
	}

	// Re-parent every surviving tag whose parent is being deleted, before the
	// delete runs. Walking the graph rather than handling only the direct
	// children of the requested tags is what makes deleting a parent and its
	// child in one batch correct: each orphan walks up to the nearest ancestor
	// that survives. Under cascade nothing matches here, because every
	// descendant is already in the delete set.
	for id, parentID := range g.parent {
		if _, deleting := deleteSet[id]; deleting || parentID == nil {
			continue
		}
		if _, parentDeleting := deleteSet[*parentID]; !parentDeleting {
			continue
		}
		var next *int64
		if childStrategy == ChildStrategyPromote {
			next = nearestSurvivingAncestor(g, deleteSet, id)
		}
		if _, err := tx.Exec(`UPDATE tags SET parent_id = ? WHERE id = ?`, next, id); err != nil {
			return fmt.Errorf("re-parent tag %d: %w", id, err)
		}
	}

	// Remove associations, then the tags.
	for id := range deleteSet {
		if _, err := tx.Exec(`DELETE FROM file_tags WHERE tag_id = ?`, id); err != nil {
			return fmt.Errorf("remove file links for tag %d: %w", id, err)
		}
		if _, err := tx.Exec(`DELETE FROM tag_aliases WHERE tag_id = ?`, id); err != nil {
			return fmt.Errorf("remove aliases for tag %d: %w", id, err)
		}
		if _, err := tx.Exec(`DELETE FROM tags WHERE id = ?`, id); err != nil {
			return fmt.Errorf("delete tag %d: %w", id, err)
		}
	}

	// Safety net scoped to this operation: nothing may still point at a tag we
	// just deleted. Pre-existing orphans elsewhere in an old vault are not this
	// transaction's business and must not block it.
	stranded, err := countStrandedChildren(tx, deleteSet)
	if err != nil {
		return err
	}
	if stranded > 0 {
		return fmt.Errorf("%d tag(s) would be left with a deleted parent; rolled back", stranded)
	}

	return tx.Commit()
}

// previewNames lists up to three names for an error message, with a count for
// the rest, so the message stays readable for a wide fan-out.
func previewNames(names []string) string {
	const max = 3
	if len(names) <= max {
		return strings.Join(names, ", ")
	}
	return fmt.Sprintf("%s, and %d more", strings.Join(names[:max], ", "), len(names)-max)
}

// MoveTag re-parents a tag and positions it within its new level.
// newParentID is nil for the top level. beforeID places the tag immediately
// before that sibling; 0, or a tag that is not a sibling of the destination,
// appends to the end — a drop that misses its gap should not fail.
//
// sort_order is rewritten across the destination level so the stored order
// matches what the tree shows; siblings keep their relative order.
func (a *App) MoveTag(tagID int64, newParentID *int64, beforeID int64) error {
	v := a.vault()
	if v.db == nil {
		return fmt.Errorf("no vault open")
	}

	tx, err := v.db.Conn().Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	g, err := loadTagGraph(tx)
	if err != nil {
		return err
	}
	if err := validateNewParentInGraph(g, tagID, newParentID); err != nil {
		return err
	}
	// A category groups other tags, so it is a parent — but it is never a child,
	// because it exists to be seen as a top-level section.
	if newParentID != nil && g.isCategory[tagID] {
		return fmt.Errorf("%q is a category and must stay at the top level", g.names[tagID])
	}

	siblings, err := siblingOrder(tx, newParentID, tagID)
	if err != nil {
		return err
	}
	order := make([]int64, 0, len(siblings)+1)
	at := len(siblings)
	if beforeID > 0 && beforeID != tagID {
		for i, id := range siblings {
			if id == beforeID {
				at = i
				break
			}
		}
	}
	order = append(order, siblings[:at]...)
	order = append(order, tagID)
	order = append(order, siblings[at:]...)

	if _, err := tx.Exec(`UPDATE tags SET parent_id = ? WHERE id = ?`, newParentID, tagID); err != nil {
		return fmt.Errorf("move tag %d: %w", tagID, err)
	}
	for i, id := range order {
		// Only write rows that actually change, so a move does not touch the whole level.
		if _, err := tx.Exec(`UPDATE tags SET sort_order = ? WHERE id = ? AND sort_order != ?`, i, id, i); err != nil {
			return fmt.Errorf("reorder tag %d: %w", id, err)
		}
	}
	return tx.Commit()
}

// siblingOrder lists the IDs at one level of the hierarchy in display order
// (sort_order, then name — the order GetTags uses), excluding excludeID.
func siblingOrder(q queryer, parentID *int64, excludeID int64) ([]int64, error) {
	// Same order GetTags uses, so the stored order and the shown order agree.
	const orderBy = ` ORDER BY sort_order ASC, name ASC`
	var (
		rows *sql.Rows
		err  error
	)
	if parentID == nil {
		rows, err = q.Query(`SELECT id FROM tags WHERE parent_id IS NULL AND id != ?`+orderBy, excludeID)
	} else {
		rows, err = q.Query(`SELECT id FROM tags WHERE parent_id = ? AND id != ?`+orderBy, *parentID, excludeID)
	}
	if err != nil {
		return nil, fmt.Errorf("load siblings: %w", err)
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan sibling: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// MergeTags folds every source tag into targetID inside one transaction and
// deletes the sources, returning what changed.
//
// Files are moved rather than de-duplicated by hand: the INSERT OR IGNORE
// relies on the PRIMARY KEY (file_id, tag_id), so a file that already carries
// the target tag is skipped instead of failing.
//
// Aliases always survive. Each source name also becomes an alias of the target,
// unconditionally, because an alias is what keeps a saved search resolving after
// the tag it named is gone (app/search.go joins tag_aliases).
//
// Children of a source move under the target: a merge replaces the source
// everywhere, in the tree included.
//
// Merging is refused across kinds (a category and a plain tag), and when the
// target lies inside a source's subtree. Categories may merge with categories.
func (a *App) MergeTags(sourceIDs []int64, targetID int64) (*db.TagMergeResult, error) {
	v := a.vault()
	if v.db == nil {
		return nil, fmt.Errorf("no vault open")
	}
	sources := uniqueIDs(sourceIDs)
	if len(sources) == 0 {
		return nil, fmt.Errorf("merge requires at least one source tag")
	}

	tx, err := v.db.Conn().Begin()
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	g, err := loadTagGraph(tx)
	if err != nil {
		return nil, err
	}
	if !g.exists(targetID) {
		return nil, fmt.Errorf("target tag %d does not exist", targetID)
	}
	for _, id := range sources {
		if !g.exists(id) {
			return nil, fmt.Errorf("source tag %d does not exist", id)
		}
		if id == targetID {
			return nil, fmt.Errorf("cannot merge tag %q into itself", g.names[id])
		}
		// Merging only works within one kind. A category into a plain tag would
		// strand its children under a tag that holds files, and a plain tag into a
		// category would move file links onto a tag that cannot carry them.
		if g.isCategory[id] != g.isCategory[targetID] {
			return nil, fmt.Errorf("cannot merge %s %q into %s %q",
				tagKind(g.isCategory[id]), g.names[id], tagKind(g.isCategory[targetID]), g.names[targetID])
		}
		// A target inside its own source's subtree would be re-parented by the
		// merge below onto a tag that is about to be its descendant.
		if g.isDescendant(id, targetID) {
			return nil, fmt.Errorf("cannot merge %q into its own descendant %q", g.names[id], g.names[targetID])
		}
	}

	result := &db.TagMergeResult{Sources: len(sources)}

	// 1. Move file links.
	for _, id := range sources {
		var total int
		if err := tx.QueryRow(`SELECT COUNT(*) FROM file_tags WHERE tag_id = ?`, id).Scan(&total); err != nil {
			return nil, fmt.Errorf("count links of tag %d: %w", id, err)
		}
		res, err := tx.Exec(`
			INSERT OR IGNORE INTO file_tags (file_id, tag_id)
			SELECT file_id, ? FROM file_tags WHERE tag_id = ?
		`, targetID, id)
		if err != nil {
			return nil, fmt.Errorf("merge tag %d files into %d: %w", id, targetID, err)
		}
		inserted := 0
		if n, err := res.RowsAffected(); err == nil {
			inserted = int(n)
		}
		result.FilesMoved += inserted
		result.Duplicates += total - inserted // files that already carried the target tag
	}

	// 2. Move aliases. Aliases always survive a merge: an alias is what keeps an
	// old search term matching after its tag disappears.
	//
	// idx_tag_aliases_alias_nocase is unique on LOWER(alias) alone, so an alias
	// names exactly one tag in the whole vault. The source's row therefore has to
	// go before the target can take the name — insert first and the unique index
	// rejects it against the source's own row.
	for _, id := range sources {
		rows, err := tx.Query(`SELECT alias FROM tag_aliases WHERE tag_id = ?`, id)
		if err != nil {
			return nil, fmt.Errorf("read aliases of tag %d: %w", id, err)
		}
		var pending []string
		for rows.Next() {
			var alias string
			if err := rows.Scan(&alias); err != nil {
				rows.Close()
				return nil, fmt.Errorf("scan alias of tag %d: %w", id, err)
			}
			pending = append(pending, alias)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, fmt.Errorf("read aliases of tag %d: %w", id, err)
		}
		rows.Close()

		if _, err := tx.Exec(`DELETE FROM tag_aliases WHERE tag_id = ?`, id); err != nil {
			return nil, fmt.Errorf("release aliases of tag %d: %w", id, err)
		}
		for _, alias := range pending {
			res, err := tx.Exec(`INSERT OR IGNORE INTO tag_aliases (tag_id, alias) VALUES (?, ?)`, targetID, alias)
			if err != nil {
				return nil, fmt.Errorf("move alias %q from tag %d: %w", alias, id, err)
			}
			if n, err := res.RowsAffected(); err == nil && n > 0 {
				result.AliasesMoved++
			} else {
				result.AliasesSkipped++
			}
		}
	}

	// 3. Adopt each source's name as an alias of the target. Only the unique
	// index can refuse — when another tag already owns that name.
	for _, id := range sources {
		name := g.names[id]
		if strings.EqualFold(name, g.names[targetID]) {
			continue // a case-only merge, e.g. "App" into "app"
		}
		res, err := tx.Exec(`INSERT OR IGNORE INTO tag_aliases (tag_id, alias) VALUES (?, ?)`, targetID, name)
		if err != nil {
			return nil, fmt.Errorf("adopt name %q as alias of tag %d: %w", name, targetID, err)
		}
		if n, err := res.RowsAffected(); err == nil && n > 0 {
			result.NamesAdopted++
		} else {
			result.AliasesSkipped++
		}
	}

	// 4. A target with no colour takes the first source colour, so merging into a
	// freshly created, uncoloured tag keeps the colour the user was sorting by.
	if g.colors[targetID] == "" {
		for _, id := range sources {
			if g.colors[id] == "" {
				continue
			}
			if _, err := tx.Exec(`UPDATE tags SET color = ? WHERE id = ?`, g.colors[id], targetID); err != nil {
				return nil, fmt.Errorf("adopt colour from tag %d for tag %d: %w", id, targetID, err)
			}
			break
		}
	}

	// 5. Re-parent the children of every source onto the target. The cycle check
	// above guarantees the target is neither a source nor below one.
	deleteSet := make(map[int64]struct{}, len(sources))
	for _, id := range sources {
		deleteSet[id] = struct{}{}
	}
	for _, id := range sortedKeys(g.parent) {
		if id == targetID || g.parent[id] == nil {
			continue
		}
		if _, deleting := deleteSet[id]; deleting {
			continue
		}
		if _, parentDeleting := deleteSet[*g.parent[id]]; !parentDeleting {
			continue
		}
		if _, err := tx.Exec(`UPDATE tags SET parent_id = ? WHERE id = ?`, targetID, id); err != nil {
			return nil, fmt.Errorf("re-parent tag %d: %w", id, err)
		}
		result.ChildrenMoved++
	}

	// 6. Delete the sources.
	for id := range deleteSet {
		if _, err := tx.Exec(`DELETE FROM file_tags WHERE tag_id = ?`, id); err != nil {
			return nil, fmt.Errorf("remove file links for tag %d: %w", id, err)
		}
		if _, err := tx.Exec(`DELETE FROM tag_aliases WHERE tag_id = ?`, id); err != nil {
			return nil, fmt.Errorf("remove aliases for tag %d: %w", id, err)
		}
		if _, err := tx.Exec(`DELETE FROM tags WHERE id = ?`, id); err != nil {
			return nil, fmt.Errorf("delete source tag %d: %w", id, err)
		}
	}

	// 7. Same scoped safety net as DeleteTagsSafely: nothing may still point at
	// a tag this merge deleted.
	stranded, err := countStrandedChildren(tx, deleteSet)
	if err != nil {
		return nil, err
	}
	if stranded > 0 {
		return nil, fmt.Errorf("%d tag(s) would be left with a deleted parent; rolled back", stranded)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit merge: %w", err)
	}

	sourceNames := make([]string, 0, len(sources))
	for _, id := range sources {
		sourceNames = append(sourceNames, g.names[id])
	}
	utils.LogInfo(
		"merged %s into %q: %d file link(s) moved, %d deduplicated, %d alias(es) moved, %d name(s) adopted, %d skipped",
		strings.Join(sourceNames, ", "), g.names[targetID],
		result.FilesMoved, result.Duplicates, result.AliasesMoved, result.NamesAdopted, result.AliasesSkipped)
	return result, nil
}

// sortedKeys returns map keys in ascending order, so a write order derived from
// a map is reproducible.
func sortedKeys[V any](m map[int64]V) []int64 {
	keys := make([]int64, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	return keys
}

// countStrandedChildren counts tags that still point at one of the given IDs.
func countStrandedChildren(q queryer, deleted map[int64]struct{}) (int, error) {
	if len(deleted) == 0 {
		return 0, nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(deleted)), ",")
	args := make([]any, 0, len(deleted))
	for _, id := range sortedKeys(deleted) {
		args = append(args, id)
	}
	var stranded int
	if err := q.QueryRow(
		`SELECT COUNT(*) FROM tags WHERE parent_id IN (`+placeholders+`)`, args...,
	).Scan(&stranded); err != nil {
		return 0, fmt.Errorf("check for stranded children: %w", err)
	}
	return stranded, nil
}

// tagKind names a tag's kind for error messages.
func tagKind(isCategory bool) string {
	if isCategory {
		return "category"
	}
	return "tag"
}

// nearestSurvivingAncestor walks up from id's parent until it finds a tag that
// is not being deleted, or reaches the top level (nil).
func nearestSurvivingAncestor(g *tagGraph, deleteSet map[int64]struct{}, id int64) *int64 {
	steps := len(g.parent) + 1 // guard against cycles already in stored data
	for p := g.parent[id]; p != nil && steps > 0; p = g.parent[*p] {
		steps--
		if _, deleting := deleteSet[*p]; !deleting {
			parent := *p
			return &parent
		}
	}
	return nil
}

func uniqueIDs(ids []int64) []int64 {
	seen := map[int64]struct{}{}
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
