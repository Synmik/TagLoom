// Aligns with generated Wails types in wailsjs/go/models.ts (db namespace)

export interface Tag {
  id: number;
  name: string;
  color: string;
  parent_id?: number;
  is_category: number;
  sort_order: number;
  created_at: string;
}

export interface TagCreate {
  name: string;
  color: string;
  parent_id?: number;
  is_category: number;
  sort_order: number;
  aliases: string;
}

export interface TagUpdate {
  id: number;
  name: string;
  color: string;
  parent_id?: number;
  is_category: number;
  sort_order: number;
  aliases: string;
}
