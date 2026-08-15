// Aligns with generated Wails types in wailsjs/go/models.ts (db namespace)

/** Supported file extensions — mirrors utils.SupportedExtensions in Go backend */
export const SUPPORTED_EXTENSIONS = new Set([
  // Images
  '.jpg', '.jpeg', '.png', '.gif', '.webp', '.bmp', '.tiff', '.tif', '.svg', '.avif', '.jxl', '.jpegxl',
  // Videos
  '.mp4', '.mov', '.avi', '.webm', '.mkv', '.wmv', '.flv', '.m4v', '.3gp', '.3g2',
  '.vob', '.ogv', '.mpg', '.mpeg', '.m2v', '.ts', '.mts', '.m2ts',
  '.asf', '.rm', '.amv', '.f4v', '.dv', '.mxf',
  // Animated (already covered above)
])

export interface FileTag {
  id: number
  name: string
  color: string
  parent_id?: number
  is_category: number
  sort_order: number
  created_at: string
}

export interface File {
  id: number
  vault_path: string
  thumbnail_path?: string
  name?: string
  notes?: string
  link?: string
  rating: number
  is_favorite: number
  folder_path: string
  filename: string
  file_size: number
  date_created: string
  date_modified: string
  indexed_at: string
  tags?: FileTag[]
}

export interface FileUpdate {
  id: number
  name: string
  notes: string
  link: string
  rating: number
  is_favorite: number
}

export interface FilePage {
  files: File[]
  total_count: number
  page: number
  limit: number
}

export interface FileFilter {
  folder_path: string
  tag_groups: number[][]  // Each group = OR; between groups = AND
  file_formats: string[]
  min_rating: number
  favorites_only: boolean
  untagged_only: boolean
}

export interface SortOpts {
  field: string
  order: string
}

export interface FileMetadata {
  filename: string
  extension: string
  format_name: string
  mime_type: string
  size_bytes: number
  date_created: string
  date_modified: string
  resolution_width: number
  resolution_height: number
  duration_seconds: number
  dominant_colors: string[]
}
