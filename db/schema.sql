-- TagLoom Database Schema
-- Stores user-editable data, structural references, and lightweight sort fields.
-- Heavy metadata (size, resolution, duration, etc.) is fetched on-demand from the filesystem.

CREATE TABLE IF NOT EXISTS files (
    id INTEGER PRIMARY KEY,
    vault_path TEXT NOT NULL UNIQUE,  -- Relative path from vault root (e.g. "photos/vacation/beach.jpg")
    thumbnail_path TEXT,              -- Absolute path to thumbnail on disk
    name TEXT,
    notes TEXT,
    link TEXT,
    rating INTEGER DEFAULT 0,
    is_favorite INTEGER DEFAULT 0,
    folder_path TEXT NOT NULL,
    filename TEXT NOT NULL DEFAULT '',
    date_created TEXT NOT NULL DEFAULT '',
    date_modified TEXT NOT NULL DEFAULT '',
    indexed_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS tags (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    color TEXT,
    parent_id INTEGER,
    is_category INTEGER DEFAULT 0,
    sort_order INTEGER DEFAULT 0,
    created_at TEXT NOT NULL,
    FOREIGN KEY (parent_id) REFERENCES tags(id)
);

-- Case-insensitive unique index for tag names ("App" and "app" are the same tag)
CREATE UNIQUE INDEX IF NOT EXISTS idx_tags_name_nocase ON tags(LOWER(name));

CREATE TABLE IF NOT EXISTS tag_aliases (
    tag_id INTEGER NOT NULL,
    alias TEXT NOT NULL,
    PRIMARY KEY (tag_id, alias),
    FOREIGN KEY (tag_id) REFERENCES tags(id)
);

-- Case-insensitive unique index for tag aliases
CREATE UNIQUE INDEX IF NOT EXISTS idx_tag_aliases_alias_nocase ON tag_aliases(LOWER(alias));

CREATE TABLE IF NOT EXISTS file_tags (
    file_id INTEGER NOT NULL,
    tag_id INTEGER NOT NULL,
    PRIMARY KEY (file_id, tag_id),
    FOREIGN KEY (file_id) REFERENCES files(id),
    FOREIGN KEY (tag_id) REFERENCES tags(id)
);

CREATE TABLE IF NOT EXISTS excluded_folders (
    id INTEGER PRIMARY KEY,
    path TEXT NOT NULL,       -- Relative path from vault root
    created_at TEXT NOT NULL
);

-- Full-Text Search virtual table (FTS5)
-- Indexes user-set name and notes. Filename search is handled via vault_path LIKE match.
CREATE VIRTUAL TABLE IF NOT EXISTS files_fts USING fts5(
    name,
    notes,
    content='files',
    content_rowid='id'
);

-- Triggers to keep FTS in sync
CREATE TRIGGER IF NOT EXISTS files_ai AFTER INSERT ON files BEGIN
    INSERT INTO files_fts(rowid, name, notes) VALUES (new.id, new.name, new.notes);
END;

CREATE TRIGGER IF NOT EXISTS files_ad AFTER DELETE ON files BEGIN
    INSERT INTO files_fts(files_fts, rowid, name, notes) VALUES('delete', old.id, old.name, old.notes);
END;

CREATE TRIGGER IF NOT EXISTS files_au AFTER UPDATE ON files BEGIN
    INSERT INTO files_fts(files_fts, rowid, name, notes) VALUES('delete', old.id, old.name, old.notes);
    INSERT INTO files_fts(rowid, name, notes) VALUES (new.id, new.name, new.notes);
END;

-- Single-column indexes
CREATE INDEX IF NOT EXISTS idx_files_folder ON files(folder_path);
CREATE INDEX IF NOT EXISTS idx_files_rating ON files(rating);
CREATE INDEX IF NOT EXISTS idx_files_favorite ON files(is_favorite);
CREATE INDEX IF NOT EXISTS idx_files_filename ON files(filename);
CREATE INDEX IF NOT EXISTS idx_file_tags_tag ON file_tags(tag_id);
CREATE INDEX IF NOT EXISTS idx_tags_parent ON tags(parent_id);
CREATE INDEX IF NOT EXISTS idx_tags_name ON tags(name);

-- Composite indexes for common filter combinations
-- Covers: folder filter + favorites toggle + rating filter + id for row lookup
CREATE INDEX IF NOT EXISTS idx_files_folder_fav_rating ON files(folder_path, is_favorite, rating, id);
-- Covers: favorites + rating without folder filter
CREATE INDEX IF NOT EXISTS idx_files_fav_rating ON files(is_favorite, rating, id);
-- Reverse lookup: get all tags for a given file
CREATE INDEX IF NOT EXISTS idx_file_tags_file ON file_tags(file_id);
