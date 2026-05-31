package app

import (
	"fmt"
)

// AddTagsToFiles adds the given tag IDs to all the given file IDs.
// Uses a transaction so partial failures don't leave the DB inconsistent.
// Duplicates are silently ignored (INSERT OR IGNORE).
func (a *App) AddTagsToFiles(fileIDs []int64, tagIDs []int64) error {
	if a.db == nil {
		return fmt.Errorf("no vault open")
	}
	if len(fileIDs) == 0 || len(tagIDs) == 0 {
		return nil
	}

	tx, err := a.db.Conn().Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("INSERT OR IGNORE INTO file_tags (file_id, tag_id) VALUES (?, ?)")
	if err != nil {
		return fmt.Errorf("prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, fileID := range fileIDs {
		for _, tagID := range tagIDs {
			_, _ = stmt.Exec(fileID, tagID)
		}
	}

	return tx.Commit()
}

// RemoveTagsFromFiles removes the given tag IDs from all the given file IDs.
func (a *App) RemoveTagsFromFiles(fileIDs []int64, tagIDs []int64) error {
	if a.db == nil {
		return fmt.Errorf("no vault open")
	}
	if len(fileIDs) == 0 || len(tagIDs) == 0 {
		return nil
	}

	tx, err := a.db.Conn().Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("DELETE FROM file_tags WHERE file_id = ? AND tag_id = ?")
	if err != nil {
		return fmt.Errorf("prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, fileID := range fileIDs {
		for _, tagID := range tagIDs {
			_, _ = stmt.Exec(fileID, tagID)
		}
	}

	return tx.Commit()
}

// SetRatingForFiles sets the given rating (0-5) on all the given file IDs.
// Rating of 0 means "unrated".
func (a *App) SetRatingForFiles(fileIDs []int64, rating int) error {
	if a.db == nil {
		return fmt.Errorf("no vault open")
	}
	if len(fileIDs) == 0 {
		return nil
	}
	if rating < 0 || rating > 5 {
		return fmt.Errorf("rating must be between 0 and 5, got %d", rating)
	}

	tx, err := a.db.Conn().Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("UPDATE files SET rating = ? WHERE id = ?")
	if err != nil {
		return fmt.Errorf("prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, fileID := range fileIDs {
		_, _ = stmt.Exec(rating, fileID)
	}

	return tx.Commit()
}

// SetFavoriteForFiles sets the favorite flag (0 = off, 1 = on) on all the given file IDs.
func (a *App) SetFavoriteForFiles(fileIDs []int64, isFavorite int) error {
	if a.db == nil {
		return fmt.Errorf("no vault open")
	}
	if len(fileIDs) == 0 {
		return nil
	}
	if isFavorite != 0 && isFavorite != 1 {
		return fmt.Errorf("is_favorite must be 0 or 1, got %d", isFavorite)
	}

	tx, err := a.db.Conn().Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("UPDATE files SET is_favorite = ? WHERE id = ?")
	if err != nil {
		return fmt.Errorf("prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, fileID := range fileIDs {
		_, _ = stmt.Exec(isFavorite, fileID)
	}

	return tx.Commit()
}
