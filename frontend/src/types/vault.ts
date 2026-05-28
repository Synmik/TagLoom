// Aligns with generated Wails types in wailsjs/go/models.ts (db & config namespaces)

export interface VaultInfo {
  path: string
  name: string
  created_at: string
  file_count: number
}

export interface VaultConfig {
  name: string
  version: number
  created_at: string
  settings: VaultSettings
}

export interface VaultSettings {
  auto_tag_by_folder: boolean
  excluded_folders: string[]
  thumbnail_size: number
  thumbnail_quality: number
  default_sort_field: string
  default_sort_order: string
  grid_thumbnail_size: string
}

export interface FolderNode {
  path: string
  name: string
  file_count: number
  children: FolderNode[]
}
