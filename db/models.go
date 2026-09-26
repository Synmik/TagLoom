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
	// Aliases is the full desired alias list; nil or empty means "no aliases".
	// A slice rather than a delimited string on purpose: an alias may legally
	// contain a comma.
	Aliases []string `json:"aliases"`
}

// TagUpdate is used when editing an existing tag. Aliases is the full desired
// list — existing rows are replaced, so nil/empty clears them.
type TagUpdate struct {
	ID         int64    `json:"id"`
	Name       string   `json:"name"`
	Color      string   `json:"color"`
	ParentID   *int64   `json:"parent_id"`
	IsCategory int      `json:"is_category"`
	SortOrder  int      `json:"sort_order"`
	Aliases    []string `json:"aliases"`
}

// TagMergeResult reports what a merge actually changed, so the UI can state it
// instead of guessing: the frontend cannot know how many source files already
// carried the target tag.
type TagMergeResult struct {
	Sources        int `json:"sources"`
	FilesMoved     int `json:"files_moved"`     // new file_tags rows on the target
	Duplicates     int `json:"duplicates"`      // source links dropped: the file already had the target
	AliasesMoved   int `json:"aliases_moved"`   // source aliases that survived on the target
	NamesAdopted   int `json:"names_adopted"`   // source names kept as target aliases
	AliasesSkipped int `json:"aliases_skipped"` // forbidden by the unique alias index
	ChildrenMoved  int `json:"children_moved"`  // source children re-parented onto the target
}

// TagUsage reports how a tag is used, so destructive UI can state exactly what
// a delete or merge will change. DescendantFiles matches the tag-tree badge
// semantics: the tag's own files plus every descendant's files (a file tagged
// with two descendants is counted per descendant, as in the sidebar).
type TagUsage struct {
	TagID           int64  `json:"tag_id"`
	Name            string `json:"name"`
	IsCategory      int    `json:"is_category"`
	DirectFiles     int    `json:"direct_files"`
	DescendantFiles int    `json:"descendant_files"`
	Children        int    `json:"children"`
	Aliases         int    `json:"aliases"`
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
	FolderPath    string    `json:"folder_path"`
	TagGroups     [][]int64 `json:"tag_groups"` // Each group = OR; between groups = AND
	FileFormats   []string  `json:"file_formats"`
	MinRating     int       `json:"min_rating"`
	FavoritesOnly bool      `json:"favorites_only"`
	UntaggedOnly  bool      `json:"untagged_only"`
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
