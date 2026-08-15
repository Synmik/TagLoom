package db

// File represents a media file in the vault.
// Stores user-editable data, structural references, and lightweight sort fields.
// Heavy metadata (resolution, duration, etc.) is fetched on-demand from the
// filesystem; FileSize is stored and populated at scan/import time.
type File struct {
	ID            int64   `json:"id"`
	VaultPath     string  `json:"vault_path"`
	ThumbnailPath *string `json:"thumbnail_path"` // nullable in DB
	Name          *string `json:"name"`           // nullable in DB
	Notes         *string `json:"notes"`          // nullable in DB
	Link          *string `json:"link"`           // nullable in DB
	Rating        int     `json:"rating"`
	IsFavorite    int     `json:"is_favorite"`
	FolderPath    string  `json:"folder_path"`
	Filename      string  `json:"filename"`
	FileSize      int64   `json:"file_size"`
	DateCreated   string  `json:"date_created"`
	DateModified  string  `json:"date_modified"`
	IndexedAt     string  `json:"indexed_at"`
	Tags          []Tag   `json:"tags,omitempty"`
}

// FileUpdate contains fields that can be updated by the user.
type FileUpdate struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Notes      string `json:"notes"`
	Link       string `json:"link"`
	Rating     int    `json:"rating"`
	IsFavorite int    `json:"is_favorite"`
}

// Tag represents a user-defined tag.
type Tag struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Color      string `json:"color"`
	ParentID   *int64 `json:"parent_id"`
	IsCategory int    `json:"is_category"`
	SortOrder  int    `json:"sort_order"`
	CreatedAt  string `json:"created_at"`
}

// TagCreate is used when creating a new tag.
type TagCreate struct {
	Name       string `json:"name"`
	Color      string `json:"color"`
	ParentID   *int64 `json:"parent_id"`
	IsCategory int    `json:"is_category"`
	SortOrder  int    `json:"sort_order"`
	Aliases    string `json:"aliases"` // comma-separated
}

// TagUpdate is used when editing an existing tag.
type TagUpdate struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Color      string `json:"color"`
	ParentID   *int64 `json:"parent_id"`
	IsCategory int    `json:"is_category"`
	SortOrder  int    `json:"sort_order"`
	Aliases    string `json:"aliases"`
}

// TagAlias represents an alternate name for a tag.
type TagAlias struct {
	TagID int64  `json:"tag_id"`
	Alias string `json:"alias"`
}

// FileTag links a file to a tag.
type FileTag struct {
	FileID int64 `json:"file_id"`
	TagID  int64 `json:"tag_id"`
}

// ExcludedFolder represents a folder skipped during indexing.
type ExcludedFolder struct {
	ID        int64  `json:"id"`
	Path      string `json:"path"`
	CreatedAt string `json:"created_at"`
}

// FileFilter holds query parameters for GetFiles.
type FileFilter struct {
	FolderPath string   `json:"folder_path"`
	TagGroups  [][]int64 `json:"tag_groups"` // Each group = OR; between groups = AND
	FileFormats   []string `json:"file_formats"`
	MinRating     int      `json:"min_rating"`
	FavoritesOnly bool     `json:"favorites_only"`
	UntaggedOnly  bool     `json:"untagged_only"`
}

// SortOpts holds sorting parameters.
type SortOpts struct {
	Field string `json:"field"` // "name", "rating", "indexed_at", "filename", "date_modified", "file_size"
	Order string `json:"order"` // "asc" or "desc"
}

// FilePage is a paginated result of files.
type FilePage struct {
	Files      []File `json:"files"`
	TotalCount int    `json:"total_count"`
	Page       int    `json:"page"`
	Limit      int    `json:"limit"`
}

// FolderNode represents a node in the vault folder tree.
type FolderNode struct {
	Path      string       `json:"path"`
	Name      string       `json:"name"`
	FileCount int          `json:"file_count"`
	Children  []FolderNode `json:"children"`
}

// VaultInfo contains information about the current vault.
type VaultInfo struct {
	Path      string `json:"path"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
	FileCount int    `json:"file_count"`
}
