-- TagLoom Database Schema
-- Stores only user-editable data and structural references.
-- Read-only metadata (filename, size, resolution, etc.) is fetched on-demand from the filesystem.

CREATE TABLE IF NOT EXISTS files (
    id INTEGER PRIMARY KEY,
    vault_path TEXT NOT NULL UNIQUE,
    thumbnail_path TEXT,
    name TEXT,
    notes TEXT,
    link TEXT,
    rating INTEGER DEFAULT 0,
    is_favorite INTEGER DEFAULT 0,
    folder_path TEXT NOT NULL,
    indexed_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS tags (
    id INTEGER PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    color TEXT,
    parent_id INTEGER,
    is_category INTEGER DEFAULT 0,
    sort_order INTEGER DEFAULT 0,
    created_at TEXT NOT NULL,
    FOREIGN KEY (parent_id) REFERENCES tags(id)
);

CREATE TABLE IF NOT EXISTS tag_aliases (
    tag_id INTEGER NOT NULL,
    alias TEXT UNIQUE NOT NULL,
    PRIMARY KEY (tag_id, alias),
    FOREIGN KEY (tag_id) REFERENCES tags(id)
);

CREATE TABLE IF NOT EXISTS file_tags (
    file_id INTEGER NOT NULL,
    tag_id INTEGER NOT NULL,
    PRIMARY KEY (file_id, tag_id),
    FOREIGN KEY (file_id) REFERENCES files(id),
    FOREIGN KEY (tag_id) REFERENCES tags(id)
);

CREATE TABLE IF NOT EXISTS excluded_folders (
    id INTEGER PRIMARY KEY,
    path TEXT NOT NULL,
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

-- Indexes
CREATE INDEX IF NOT EXISTS idx_files_folder ON files(folder_path);
CREATE INDEX IF NOT EXISTS idx_files_rating ON files(rating);
CREATE INDEX IF NOT EXISTS idx_files_favorite ON files(is_favorite);
CREATE INDEX IF NOT EXISTS idx_file_tags_tag ON file_tags(tag_id);
CREATE INDEX IF NOT EXISTS idx_tags_parent ON tags(parent_id);
CREATE INDEX IF NOT EXISTS idx_tags_name ON tags(name);
