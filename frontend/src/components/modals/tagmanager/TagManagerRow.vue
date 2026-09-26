<template>
  <div
    class="tm-row"
    :class="{
      'is-selected': selected,
      'is-cursor': cursor,
      'is-dim': dim,
      'is-unused': unused && !selected && !cursor,
    }"
    :style="{ paddingLeft: `${8 + depth * 16}px` }"
    @click="$emit('select', $event)"
    @dblclick="$emit('open')"
  >
    <span
      class="tm-caret"
      :title="hasChildren ? 'Expand / collapse' : ''"
      @click.stop="$emit('toggle')"
    >
      <ChevronDown v-if="hasChildren && expanded" :size="11" />
      <ChevronRight v-else-if="hasChildren" :size="11" />
    </span>
    <span class="tm-dot" :style="{ background: tag.color || '#666' }"></span>
    <span class="tm-name" :class="{ 'is-category': tag.is_category === 1 }">{{ tag.name }}</span>
    <span v-if="aliasText" class="tm-alias" :title="`Aliases: ${aliases.join(', ')}`">
      {{ aliasText }}
    </span>
    <span v-if="tag.is_category === 1" class="tm-badge">cat</span>
    <span class="tm-flex"></span>
    <span class="tm-count" :title="countTitle">{{ count }}</span>
    <span class="tm-actions">
      <button
        class="tm-action"
        title="Move up in this level"
        :disabled="!canMoveUp"
        @click.stop="$emit('move-up')"
      >
        <ArrowUp :size="12" />
      </button>
      <button
        class="tm-action"
        title="Move down in this level"
        :disabled="!canMoveDown"
        @click.stop="$emit('move-down')"
      >
        <ArrowDown :size="12" />
      </button>
      <button class="tm-action" title="Edit in the details pane" @click.stop="$emit('edit')">
        <Pencil :size="12" />
      </button>
      <button class="tm-action is-danger" title="Delete tag" @click.stop="$emit('delete')">
        <Trash2 :size="12" />
      </button>
    </span>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { ArrowDown, ArrowUp, ChevronDown, ChevronRight, Pencil, Trash2 } from "@lucide/vue";
import type { Tag } from "../../../types/tag";

const props = withDefaults(
  defineProps<{
    tag: Tag;
    /** Indent level, 0 for the top level. */
    depth: number;
    expanded: boolean;
    hasChildren: boolean;
    selected: boolean;
    /** Keyboard cursor — the row arrow keys point at. */
    cursor: boolean;
    /** Search is active and this row is shown only as context (ancestor/descendant). */
    dim?: boolean;
    canMoveUp?: boolean;
    canMoveDown?: boolean;
    /** Direct file links; a category shows links of its children instead. */
    count?: number;
    /** Aliases of this tag — shown as a chip so they are manageable at a glance. */
    aliases?: string[];
    /** True when the tag's parent chain loops back on itself (corrupt data). */
    cyclic?: boolean;
    /** No file carries this tag — dimmed so the used vocabulary stands out. */
    unused?: boolean;
  }>(),
  {
    dim: false,
    canMoveUp: true,
    canMoveDown: true,
    count: 0,
    aliases: () => [],
    cyclic: false,
    unused: false,
  },
);

defineEmits<{
  select: [ev: MouseEvent];
  /** Double click / Enter — filter the gallery by this tag. */
  open: [];
  toggle: [];
  "move-up": [];
  "move-down": [];
  /** Select this row and focus the name field in the details pane. */
  edit: [];
  delete: [];
}>();

const aliasText = computed(() => {
  if (props.aliases.length === 0) return "";
  if (props.aliases.length === 1) return props.aliases[0];
  return `${props.aliases[0]} +${props.aliases.length - 1}`;
});

const countTitle = computed(() =>
  props.tag.is_category === 1
    ? `${props.count} files under this category`
    : `${props.count} file(s) tagged`,
);
</script>

<style scoped>
.tm-row {
  display: flex;
  align-items: center;
  gap: 6px;
  padding-right: 8px;
  height: 100%;
  border-radius: 6px;
  cursor: pointer;
  font-size: 13px;
  color: #ccc;
  user-select: none;
}
.tm-row:hover {
  background: #1e1e1e;
}
.tm-row.is-selected {
  background: #14532d;
  color: #e8e8e8;
}
.tm-row.is-cursor {
  outline: 1px solid #2f6f43;
  outline-offset: -1px;
}
.tm-row.is-dim {
  opacity: 0.45;
}
/* Nothing carries this tag yet — kept visible, kept quiet. */
.tm-row.is-unused .tm-name,
.tm-row.is-unused .tm-count {
  opacity: 0.5;
}
.tm-caret {
  width: 12px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #666;
}
.tm-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}
.tm-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 40%;
}
.tm-name.is-category {
  font-style: italic;
  color: #999;
}
.tm-alias {
  font-size: 10px;
  color: #777;
  background: #1a1a1a;
  border-radius: 4px;
  padding: 1px 5px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 120px;
  flex-shrink: 0;
}
.tm-badge,
.tm-count {
  font-size: 9px;
  color: #555;
  background: #1a1a1a;
  padding: 1px 5px;
  border-radius: 4px;
  flex-shrink: 0;
}
.tm-badge {
  text-transform: uppercase;
  letter-spacing: 0.5px;
}
.tm-count {
  font-size: 11px;
  padding: 1px 6px;
  min-width: 26px;
  text-align: right;
}
.tm-flex {
  flex: 1;
}
.tm-actions {
  display: none;
  gap: 2px;
  flex-shrink: 0;
}
.tm-row:hover .tm-actions,
.tm-row.is-selected .tm-actions {
  display: flex;
}
.tm-action {
  background: none;
  border: none;
  color: #777;
  padding: 2px;
  border-radius: 4px;
  cursor: pointer;
  display: flex;
  align-items: center;
}
.tm-action:hover:not(:disabled) {
  color: #e8e8e8;
  background: #2a2a2a;
}
.tm-action:disabled {
  opacity: 0.25;
  cursor: default;
}
.tm-action.is-danger:hover:not(:disabled) {
  color: #ef4444;
}
</style>
