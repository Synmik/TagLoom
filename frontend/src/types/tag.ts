export interface Tag {
  id: number
  name: string
  color: string
  parent_id: number | null
  is_category: number
  sort_order: number
  created_at: string
}

export interface TagCreate {
  name: string
  color: string
  parent_id: number | null
  is_category: number
  sort_order: number
  aliases: string
}

export interface TagUpdate {
  id: number
  name: string
  color: string
  parent_id: number | null
  is_category: number
  sort_order: number
  aliases: string
}

export interface TagAlias {
  tag_id: number
  alias: string
}

export interface TagCategory {
  name: string
  tags: Tag[]
}
