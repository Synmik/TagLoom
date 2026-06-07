package app

import (
	"fmt"
	"strings"
	"time"

	"TagLoom/db"
)

// GetTags returns all tags, optionally filtered by category.
func (a *App) GetTags(category string) ([]db.Tag, error) {
	if a.db == nil {
		return nil, fmt.Errorf("no vault open")
	}

	query := `
		SELECT id, name, color, parent_id, is_category, sort_order, created_at
		FROM tags
	`
	if category != "" {
		// TODO: Implement category filtering
	}
	query += ` ORDER BY sort_order ASC, name ASC`

	rows, err := a.db.Conn().Query(query)
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
	if a.db == nil {
		return nil, fmt.Errorf("no vault open")
	}

	// Check for case-insensitive duplicate
	var existing db.Tag
	err := a.db.Conn().QueryRow(`
		SELECT id, name, color, parent_id, is_category, sort_order, created_at
		FROM tags WHERE LOWER(name) = LOWER(?)
	`, tag.Name).Scan(&existing.ID, &existing.Name, &existing.Color, &existing.ParentID,
		&existing.IsCategory, &existing.SortOrder, &existing.CreatedAt)
	if err == nil {
		// Tag with this name (case-insensitive) already exists — return it
		return &existing, nil
	}

	now := time.Now().Format(time.RFC3339)
	result, err := a.db.Conn().Exec(`
		INSERT INTO tags (name, color, parent_id, is_category, sort_order, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, tag.Name, tag.Color, tag.ParentID, tag.IsCategory, tag.SortOrder, now)
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()

	// Insert aliases
	if tag.Aliases != "" {
		for _, alias := range strings.Split(tag.Aliases, ",") {
			alias = strings.TrimSpace(alias)
			if alias != "" {
				a.db.Conn().Exec(`
					INSERT OR IGNORE INTO tag_aliases (tag_id, alias) VALUES (?, ?)
				`, id, alias)
			}
		}
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
// (ignoring case) returns an error.
func (a *App) UpdateTag(tag *db.TagUpdate) error {
	if a.db == nil {
		return fmt.Errorf("no vault open")
	}

	// Check for case-insensitive duplicate (excluding self)
	var count int
	err := a.db.Conn().QueryRow(`
		SELECT COUNT(*) FROM tags WHERE LOWER(name) = LOWER(?) AND id != ?
	`, tag.Name, tag.ID).Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("tag name %q already exists (case-insensitive)", tag.Name)
	}

	_, err = a.db.Conn().Exec(`
		UPDATE tags SET
			name = ?, color = ?, parent_id = ?, is_category = ?, sort_order = ?
		WHERE id = ?
	`, tag.Name, tag.Color, tag.ParentID, tag.IsCategory, tag.SortOrder, tag.ID)
	if err != nil {
		return err
	}

	// Update aliases: remove old, insert new
	a.db.Conn().Exec("DELETE FROM tag_aliases WHERE tag_id = ?", tag.ID)
	if tag.Aliases != "" {
		for _, alias := range strings.Split(tag.Aliases, ",") {
			alias = strings.TrimSpace(alias)
			if alias != "" {
				a.db.Conn().Exec(`
					INSERT OR IGNORE INTO tag_aliases (tag_id, alias) VALUES (?, ?)
				`, tag.ID, alias)
			}
		}
	}
	return nil
}

// DeleteTag removes a tag and its aliases. File associations are also removed.
func (a *App) DeleteTag(id int64) error {
	if a.db == nil {
		return fmt.Errorf("no vault open")
	}

	tx, _ := a.db.Conn().Begin()
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
func (a *App) AddTagToFile(fileID, tagID int64) error {
	if a.db == nil {
		return fmt.Errorf("no vault open")
	}
	_, err := a.db.Conn().Exec(`
		INSERT OR IGNORE INTO file_tags (file_id, tag_id) VALUES (?, ?)
	`, fileID, tagID)
	return err
}

// RemoveTagFromFile disassociates a tag from a file.
func (a *App) RemoveTagFromFile(fileID, tagID int64) error {
	if a.db == nil {
		return fmt.Errorf("no vault open")
	}
	_, err := a.db.Conn().Exec(`
		DELETE FROM file_tags WHERE file_id = ? AND tag_id = ?
	`, fileID, tagID)
	return err
}

// GetTagFileCount returns the number of files associated with a tag.
func (a *App) GetTagFileCount(tagID int64) (int, error) {
	if a.db == nil {
		return 0, fmt.Errorf("no vault open")
	}

	var count int
	err := a.db.Conn().QueryRow(`
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
	if a.db == nil {
		return nil, fmt.Errorf("no vault open")
	}

	counts := make(map[int64]int)

	rows, err := a.db.Conn().Query(`
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
	if a.db == nil {
		return nil, fmt.Errorf("no vault open")
	}

	rows, err := a.db.Conn().Query(`
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
	if a.db == nil {
		return nil, fmt.Errorf("no vault open")
	}

	rows, err := a.db.Conn().Query(`
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
