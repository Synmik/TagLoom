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
  /** Full desired alias list; empty means "no aliases". An alias may contain a comma. */
  aliases: string[];
}

export interface TagUpdate {
  id: number;
  name: string;
  color: string;
  parent_id?: number;
  is_category: number;
  sort_order: number;
  /** Full desired alias list — replaces the stored one, so empty clears aliases. */
  aliases: string[];
}

/** Mirrors db.TagUsage in wailsjs/go/models.ts — how a tag is used, for
 *  safe-delete and merge previews. */
export interface TagUsage {
  tag_id: number;
  direct_files: number;
  /** Direct files plus every file carried by a descendant of the tag. */
  descendant_files: number;
  children: number;
  aliases: number;
  name: string;
  is_category: number;
}

/** Mirrors db.TagMergeResult — what a merge actually changed. */
export interface TagMergeResult {
  sources: number;
  files_moved: number;
  /** Source links dropped because the file already carried the target. */
  duplicates: number;
  aliases_moved: number;
  /** Source names kept on the target as aliases, so old searches still resolve. */
  names_adopted: number;
  aliases_skipped: number;
  children_moved: number;
}

/** What DeleteTagsSafely does with the children of a deleted tag. */
export type ChildStrategy = "promote" | "root" | "cascade" | "block";

/** A tag found by matching an alias rather than a name. */
export interface AliasHit {
  tag: Tag;
  alias: string;
}

/** Mirrors db.TagAlias — one row of the global unique alias index. */
export interface TagAlias {
  tag_id: number;
  alias: string;
}
