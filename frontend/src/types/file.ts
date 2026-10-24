// Aligns with generated Wails types in wailsjs/go/models.ts (db namespace)

/** Supported file extensions — mirrors utils.SupportedExtensions in Go backend */
export const SUPPORTED_EXTENSIONS = new Set([
  // Images
  ".jpg",
  ".jpeg",
  ".png",
  ".gif",
  ".webp",
  ".bmp",
  ".tiff",
  ".tif",
  ".svg",
  ".avif",
  ".jxl",
  ".jpegxl",
  // Videos
  ".mp4",
  ".mov",
  ".avi",
  ".webm",
  ".mkv",
  ".wmv",
  ".flv",
  ".m4v",
  ".3gp",
  ".3g2",
  ".vob",
  ".ogv",
  ".mpg",
  ".mpeg",
  ".m2v",
  ".ts",
  ".mts",
  ".m2ts",
  ".asf",
  ".rm",
  ".amv",
  ".f4v",
  ".dv",
  ".mxf",
  // Animated (already covered above)
  // Documents — first page / embedded cover is thumbnailed by the backend, and
  // Markdown and text are read inline
  ".pdf",
  ".epub",
  ".md",
  ".txt",
]);

export interface FileTag {
  id: number;
  name: string;
  color: string;
  parent_id?: number;
  is_category: number;
  sort_order: number;
  created_at: string;
}

export interface File {
  id: number;
  vault_path: string;
  thumbnail_path?: string;
  name?: string;
  notes?: string;
  link?: string;
  rating: number;
  is_favorite: number;
  folder_path: string;
  filename: string;
  file_size: number;
  date_created: string;
  date_modified: string;
  indexed_at: string;
  tags?: FileTag[];
}

export interface FileUpdate {
  id: number;
  name: string;
  notes: string;
  link: string;
  rating: number;
  is_favorite: number;
}

export interface FilePage {
  files: File[];
  total_count: number;
  page: number;
  limit: number;
}

export interface FileFilter {
  folder_path: string;
  tag_groups: number[][]; // Each group = OR; between groups = AND
  file_formats: string[];
  min_rating: number;
  favorites_only: boolean;
  untagged_only: boolean;
  search_query: string; // free text: names/notes (FTS5), paths, tag names/aliases
}

export interface SortOpts {
  field: string;
  order: string;
}

export interface FileMetadata {
  filename: string;
  extension: string;
  format_name: string;
  mime_type: string;
  /** Backend category: image | animated | video | document (utils.GetFileCategory) */
  category: string;
  size_bytes: number;
  date_created: string;
  date_modified: string;
  resolution_width: number;
  resolution_height: number;
  duration_seconds: number;
  dominant_colors: string[];
  page_count: number; // PDF pages, 0 for other formats
  title: string; // document title (EPUB package metadata)
  author: string; // document author (EPUB package metadata)
  line_count: number; // .md / .txt only, 0 elsewhere
  word_count: number; // .md / .txt only, 0 elsewhere
  /** True when the file is bigger than the backend counted: both tallies are floors */
  text_truncated: boolean;
}

/**
 * Head of a .md or .txt file, decoded to UTF-8 by the backend — the inline
 * preview text. `text` never exceeds 256 KiB of the file.
 */
export interface TextContent {
  file_id: number;
  text: string;
  /** utf-8 | utf-16le | utf-16be, from the file's BOM */
  encoding: string;
  bytes_read: number;
  truncated: boolean;
}
