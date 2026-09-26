package app

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"TagLoom/db"
	"TagLoom/utils"
)

// execer is satisfied by both *sql.DB and *sql.Tx, so alias helpers can run
// either outside a transaction (plain tag create/update) or inside one
// (merge), without branching on the concrete type.
type execer interface {
	Exec(query string, args ...any) (sql.Result, error)
}

// cleanAliases normalizes a caller-supplied alias list: trims whitespace,
// drops empties, and removes case-insensitive duplicates (tag aliases are
// unique case-insensitively — idx_tag_aliases_alias_nocase).
func cleanAliases(aliases []string) []string {
	cleaned := make([]string, 0, len(aliases))
	seen := make(map[string]struct{}, len(aliases))
	for _, alias := range aliases {
		alias = strings.TrimSpace(alias)
		if alias == "" {
			continue
		}
		key := strings.ToLower(alias)
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		cleaned = append(cleaned, alias)
	}
	return cleaned
}

// replaceAliases sets a tag's aliases to exactly the given list, returning how
// many were stored and how many were dropped because another tag already owns
// that alias (the case-insensitive unique index forbids the duplicate).
//
// An alias collision is expected and reported, not an error. A genuine DB
// failure is returned so callers that must be atomic (merge) can roll back;
// callers where aliases are secondary (create/update) log it instead.
func replaceAliases(db execer, tagID int64, aliases []string) (stored int, skipped int, err error) {
	if _, err := db.Exec("DELETE FROM tag_aliases WHERE tag_id = ?", tagID); err != nil {
		return 0, 0, fmt.Errorf("clear aliases: %w", err)
	}
	for _, alias := range cleanAliases(aliases) {
		res, err := db.Exec(`INSERT OR IGNORE INTO tag_aliases (tag_id, alias) VALUES (?, ?)`, tagID, alias)
		if err != nil {
			return stored, skipped, fmt.Errorf("insert alias %q: %w", alias, err)
		}
		if n, err := res.RowsAffected(); err == nil && n == 0 {
			skipped++
		} else {
			stored++
		}
	}
	return stored, skipped, nil
}

// GetTags returns all tags, optionally filtered by category.
func (a *App) GetTags(_ string) ([]db.Tag, error) { // parameter reserved for category filtering (TODO)
	v := a.vault()
	if v.db == nil {
		return nil, fmt.Errorf("no vault open")
	}

	query := `
		SELECT id, name, color, parent_id, is_category, sort_order, created_at
		FROM tags
	`
	// TODO: Implement category filtering
	query += ` ORDER BY sort_order ASC, name ASC`

	rows, err := v.db.Conn().Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []db.Tag
	for rows.Next() {
		var t db.Tag
		if err := rows.Scan(&t.ID, &t.Name, &t.Color, &t.ParentID,
			&t.IsCategory, &t.SortOrder, &t.CreatedAt); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, nil
}

// CreateTag creates a new tag.
// Tag names are case-insensitive: "App" and "app" are treated as the same tag.
func (a *App) CreateTag(tag *db.TagCreate) (*db.Tag, error) {
	v := a.vault()
	if v.db == nil {
		return nil, fmt.Errorf("no vault open")
	}

	// Check for case-insensitive duplicate
	var existing db.Tag
	err := v.db.Conn().QueryRow(`
		SELECT id, name, color, parent_id, is_category, sort_order, created_at
		FROM tags WHERE LOWER(name) = LOWER(?)
	`, tag.Name).Scan(&existing.ID, &existing.Name, &existing.Color, &existing.ParentID,
		&existing.IsCategory, &existing.SortOrder, &existing.CreatedAt)
	if err == nil {
		// Tag with this name (case-insensitive) already exists — return it
		return &existing, nil
	}

	now := time.Now().Format(time.RFC3339)
	result, err := v.db.Conn().Exec(`
		INSERT INTO tags (name, color, parent_id, is_category, sort_order, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, tag.Name, tag.Color, tag.ParentID, tag.IsCategory, tag.SortOrder, now)
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()

	// Aliases are best-effort: a DB hiccup on an optional alias must not fail
	// tag creation, so it is logged rather than returned.
	if _, _, err := replaceAliases(v.db.Conn(), id, tag.Aliases); err != nil {
		utils.LogWarn("create tag %d: aliases: %v", id, err)
	}

	return &db.Tag{
		ID:         id,
		Name:       tag.Name,
		Color:      tag.Color,
		ParentID:   tag.ParentID,
		IsCategory: tag.IsCategory,
		SortOrder:  tag.SortOrder,
		CreatedAt:  now,
	}, nil
}

// UpdateTag updates an existing tag.
// Tag name changes are case-insensitive: renaming to a name that already exists
// (ignoring case) returns an error. A parent that is the tag itself, one of its
// own descendants, or a tag that does not exist is rejected — either would leave
// the subtree unreachable in the tag tree.
func (a *App) UpdateTag(tag *db.TagUpdate) error {
	v := a.vault()
	if v.db == nil {
		return fmt.Errorf("no vault open")
	}

	if err := a.validateNewParent(v.db.Conn(), tag.ID, tag.ParentID); err != nil {
		return err
	}
	// A category is a top-level grouping tag, so its parent is cleared on save
	// rather than rejected. Rejecting would lock the rows that vaults created
	// before this rule may already contain — a category with a parent could not
	// even be renamed. Migration 7 flattens those rows for new databases.
	if tag.IsCategory == 1 {
		tag.ParentID = nil
	}

	// Check for case-insensitive duplicate (excluding self)
	var count int
	err := v.db.Conn().QueryRow(`
		SELECT COUNT(*) FROM tags WHERE LOWER(name) = LOWER(?) AND id != ?
	`, tag.Name, tag.ID).Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("tag name %q already exists (case-insensitive)", tag.Name)
	}

	_, err = v.db.Conn().Exec(`
		UPDATE tags SET
			name = ?, color = ?, parent_id = ?, is_category = ?, sort_order = ?
		WHERE id = ?
	`, tag.Name, tag.Color, tag.ParentID, tag.IsCategory, tag.SortOrder, tag.ID)
	if err != nil {
		return err
	}

	// Replace aliases with the submitted list (empty list clears them).
	// Best-effort, as in CreateTag.
	if _, _, err := replaceAliases(v.db.Conn(), tag.ID, tag.Aliases); err != nil {
		utils.LogWarn("update tag %d: aliases: %v", tag.ID, err)
	}
	return nil
}

// DeleteTag removes a tag and its aliases. File associations are also removed.
//
// Deprecated: this leaves child tags pointing at a row that no longer exists —
// PRAGMA foreign_keys is off, so SQLite will not object. Use DeleteTagsSafely,
// which promotes, roots, or cascades the children and checks that nothing is
// stranded. Kept only because it is an existing published binding.
func (a *App) DeleteTag(id int64) error {
	v := a.vault()
	if v.db == nil {
		return fmt.Errorf("no vault open")
	}

	tx, _ := v.db.Conn().Begin()
	defer tx.Rollback()

	_, err := tx.Exec("DELETE FROM tag_aliases WHERE tag_id = ?", id)
	if err != nil {
		return err
	}
	_, err = tx.Exec("DELETE FROM file_tags WHERE tag_id = ?", id)
	if err != nil {
		return err
	}
	_, err = tx.Exec("DELETE FROM tags WHERE id = ?", id)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// AddTagToFile associates a tag with a file.
// Category tags (is_category=1) cannot be assigned to files.
func (a *App) AddTagToFile(fileID, tagID int64) error {
	v := a.vault()
	if v.db == nil {
		return fmt.Errorf("no vault open")
	}

	// Prevent assigning category tags to files
	var isCategory int
	err := v.db.Conn().QueryRow("SELECT is_category FROM tags WHERE id = ?", tagID).Scan(&isCategory)
	if err != nil {
		return fmt.Errorf("tag not found: %w", err)
	}
	if isCategory == 1 {
		return fmt.Errorf("cannot assign category tag to a file")
	}

	_, err = v.db.Conn().Exec(`
		INSERT OR IGNORE INTO file_tags (file_id, tag_id) VALUES (?, ?)
	`, fileID, tagID)
	return err
}

// RemoveTagFromFile disassociates a tag from a file.
func (a *App) RemoveTagFromFile(fileID, tagID int64) error {
	v := a.vault()
	if v.db == nil {
		return fmt.Errorf("no vault open")
	}
	_, err := v.db.Conn().Exec(`
		DELETE FROM file_tags WHERE file_id = ? AND tag_id = ?
	`, fileID, tagID)
	return err
}

// GetTagFileCount returns the number of files associated with a tag.
func (a *App) GetTagFileCount(tagID int64) (int, error) {
	v := a.vault()
	if v.db == nil {
		return 0, fmt.Errorf("no vault open")
	}

	var count int
	err := v.db.Conn().QueryRow(`
		SELECT COUNT(*) FROM file_tags WHERE tag_id = ?
	`, tagID).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// GetAllTagFileCounts returns a map of tag_id -> file_count for all tags.
// More efficient than calling GetTagFileCount per tag.
func (a *App) GetAllTagFileCounts() (map[int64]int, error) {
	v := a.vault()
	if v.db == nil {
		return nil, fmt.Errorf("no vault open")
	}

	counts := make(map[int64]int)

	rows, err := v.db.Conn().Query(`
		SELECT tag_id, COUNT(*) FROM file_tags GROUP BY tag_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var tagID int64
		var count int
		if err := rows.Scan(&tagID, &count); err != nil {
			return nil, err
		}
		counts[tagID] = count
	}
	return counts, nil
}

// GetTagAliases returns all aliases for a given tag.
func (a *App) GetTagAliases(tagID int64) ([]string, error) {
	v := a.vault()
	if v.db == nil {
		return nil, fmt.Errorf("no vault open")
	}

	rows, err := v.db.Conn().Query(`
		SELECT alias FROM tag_aliases WHERE tag_id = ? ORDER BY alias ASC
	`, tagID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var aliases []string
	for rows.Next() {
		var alias string
		if err := rows.Scan(&alias); err != nil {
			return nil, err
		}
		aliases = append(aliases, alias)
	}
	return aliases, nil
}

// GetFileTags returns all tags associated with a file.
func (a *App) GetFileTags(fileID int64) ([]db.Tag, error) {
	v := a.vault()
	if v.db == nil {
		return nil, fmt.Errorf("no vault open")
	}

	rows, err := v.db.Conn().Query(`
		SELECT t.id, t.name, t.color, t.parent_id, t.is_category, t.sort_order, t.created_at
		FROM tags t
		JOIN file_tags ft ON t.id = ft.tag_id
		WHERE ft.file_id = ?
		ORDER BY t.sort_order ASC, t.name ASC
	`, fileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []db.Tag
	for rows.Next() {
		var t db.Tag
		if err := rows.Scan(&t.ID, &t.Name, &t.Color, &t.ParentID,
			&t.IsCategory, &t.SortOrder, &t.CreatedAt); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, nil
}
