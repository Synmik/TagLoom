<template>
  <Teleport to="body">
    <div class="dlg-overlay" @click.self="$emit('cancel')">
      <div class="dlg">
        <div class="dlg-header">
          <h3>Delete {{ tags.length }} {{ label }}?</h3>
          <button class="dlg-close" @click="$emit('cancel')"><X :size="16" /></button>
        </div>

        <div class="dlg-body">
          <ul class="dlg-list">
            <li v-for="tag in tags" :key="tag.id">
              <span class="dlg-dot" :style="{ background: tag.color || '#666' }"></span>
              <span class="dlg-name">{{ tag.name }}</span>
              <span class="dlg-meta">{{ meta(tag) }}</span>
            </li>
          </ul>

          <div class="dlg-warning">
            Removes tags from <b>{{ totals.files }}</b> file(s). The files themselves are not
            touched.
          </div>

          <!-- Merging is the path that keeps every file link, so offer it. -->
          <button v-if="totals.files > 0" class="dlg-link" @click="$emit('mergeInstead')">
            Merge into another {{ tags.length === 1 ? "tag" : "of these tags" }} instead…
          </button>

          <template v-if="totals.children > 0">
            <div class="dlg-label">What happens to their child tags</div>
            <ul class="dlg-children">
              <li v-for="line in childLines" :key="line.id">
                <span>{{ line.name }}</span>
                <ArrowRight :size="11" class="dlg-arrow" />
                <span class="dlg-child-names">{{ line.children }}</span>
              </li>
            </ul>
            <div class="dlg-radios">
              <label v-for="option in strategyOptions" :key="option.value">
                <input v-model="strategy" type="radio" :value="option.value" />
                <span>{{ option.label }}</span>
                <em>{{ option.hint }}</em>
              </label>
            </div>
          </template>
        </div>

        <div class="dlg-footer">
          <button class="dlg-cancel" @click="$emit('cancel')">Cancel</button>
          <button class="dlg-delete" :class="{ 'is-armed': confirming }" @click="onConfirm">
            <template v-if="!confirming">{{ confirmLabel }}</template>
            <template v-else>Click again to delete permanently</template>
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from "vue";
import { ArrowRight, X } from "@lucide/vue";
import { useTagsStore } from "../../../stores/tags";
import type { ChildStrategy, Tag, TagUsage } from "../../../types/tag";

const props = defineProps<{
  /** Tags about to be deleted. */
  tags: Tag[];
}>();

const emit = defineEmits<{
  cancel: [];
  confirm: [strategy: ChildStrategy];
  /** The user chose to fold these tags into another one instead of deleting. */
  mergeInstead: [];
}>();

const tagsStore = useTagsStore();

const usage = ref<Record<number, TagUsage>>({});
const strategy = ref<ChildStrategy>("promote");
const confirming = ref(false);

const label = computed(() => (props.tags.length === 1 ? "tag" : "tags"));

const totals = computed(() => {
  let files = 0;
  let children = 0;
  for (const tag of props.tags) {
    const u = usage.value[tag.id];
    if (!u) continue;
    files += u.direct_files;
    children += u.children;
  }
  return { files, children };
});

const confirmLabel = computed(() =>
  totals.value.files === 0
    ? `Delete ${label.value}`
    : `Delete — removes tags from ${totals.value.files} file(s)`,
);

const strategyOptions: { value: ChildStrategy; label: string; hint: string }[] = [
  {
    value: "promote",
    label: "Move up one level",
    hint: "children take the deleted tag's place",
  },
  { value: "root", label: "Move to top level", hint: "children become roots" },
  { value: "cascade", label: "Delete them too", hint: "removes the whole subtree" },
  { value: "block", label: "Refuse", hint: "only delete tags without children" },
];

// Which children go where, so the single strategy choice is an informed one.
const childLines = computed(() => {
  const lines: { id: number; name: string; children: string }[] = [];
  for (const tag of props.tags) {
    const kids = tagsStore.tags.filter((t) => t.parent_id === tag.id);
    if (kids.length === 0) continue;
    const names = kids.map((k) => k.name);
    const shown = names.slice(0, 4).join(", ");
    lines.push({
      id: tag.id,
      name: tag.name,
      children: names.length > 4 ? `${shown} +${names.length - 4} more` : shown,
    });
  }
  return lines;
});

function meta(tag: Tag): string {
  const u = usage.value[tag.id];
  if (!u) return "";
  const parts = [`${u.direct_files} files`];
  if (u.descendant_files > u.direct_files) {
    parts.push(`${u.descendant_files} incl. children`);
  }
  if (u.aliases > 0) parts.push(`${u.aliases} aliases`);
  if (u.children > 0) parts.push(`${u.children} children`);
  return parts.join(" · ");
}

function onConfirm() {
  if (!confirming.value) {
    confirming.value = true;
    return;
  }
  emit("confirm", strategy.value);
}

// Escape closes this dialog only, not the Tag Manager behind it.
function onKeydown(e: KeyboardEvent) {
  if (e.key !== "Escape") return;
  e.stopPropagation();
  emit("cancel");
}

onMounted(() => {
  window.addEventListener("keydown", onKeydown, true);
  void tagsStore
    .fetchTagUsage(props.tags.map((t) => t.id))
    .then((u) => (usage.value = { ...usage.value, ...u }));
});

onBeforeUnmount(() => window.removeEventListener("keydown", onKeydown, true));
</script>

<style scoped>
.dlg-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.7);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 200;
}
.dlg {
  background: #111;
  border: 1px solid #222;
  border-radius: 12px;
  width: 480px;
  max-height: 85vh;
  display: flex;
  flex-direction: column;
  box-shadow: 0 16px 48px rgba(0, 0, 0, 0.6);
}
.dlg-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 14px 18px;
  border-bottom: 1px solid #1a1a1a;
}
.dlg-header h3 {
  margin: 0;
  color: #e8e8e8;
  font-size: 14px;
  font-weight: 600;
  font-family: "Inter", sans-serif;
}
.dlg-close {
  background: none;
  border: none;
  color: #666;
  cursor: pointer;
  padding: 4px;
  border-radius: 4px;
}
.dlg-close:hover {
  color: #e8e8e8;
  background: #1a1a1a;
}
.dlg-body {
  padding: 16px 18px;
  display: flex;
  flex-direction: column;
  gap: 12px;
  overflow-y: auto;
}
.dlg-list {
  list-style: none;
  margin: 0;
  padding: 0;
  max-height: 130px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 3px;
}
.dlg-list li {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: #ccc;
  font-family: "Inter", sans-serif;
}
.dlg-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.dlg-meta {
  color: #666;
  font-size: 11px;
  flex-shrink: 0;
}
.dlg-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}
.dlg-warning {
  font-size: 12px;
  color: #f59e0b;
  font-family: "Inter", sans-serif;
}
.dlg-warning b {
  color: #fbbf24;
}
.dlg-link {
  background: none;
  border: none;
  color: #4ade80;
  font-size: 12px;
  font-family: "Inter", sans-serif;
  text-align: left;
  padding: 0;
  cursor: pointer;
}
.dlg-link:hover {
  text-decoration: underline;
}
.dlg-label {
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  color: #666;
  font-family: "Inter", sans-serif;
}
.dlg-children {
  list-style: none;
  margin: 0;
  padding: 0;
  max-height: 90px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.dlg-children li {
  display: flex;
  align-items: center;
  gap: 5px;
  font-size: 11px;
  color: #999;
  font-family: "Inter", sans-serif;
}
.dlg-arrow {
  color: #555;
  flex-shrink: 0;
}
.dlg-child-names {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.dlg-radios {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.dlg-radios label {
  display: flex;
  align-items: baseline;
  gap: 8px;
  font-size: 12px;
  color: #ccc;
  font-family: "Inter", sans-serif;
  cursor: pointer;
}
.dlg-radios em {
  color: #666;
  font-style: normal;
  font-size: 11px;
}
.dlg-footer {
  display: flex;
  gap: 8px;
  padding: 12px 18px;
  border-top: 1px solid #1a1a1a;
}
.dlg-cancel {
  flex: 1;
  background: #1a1a1a;
  color: #ccc;
  border: 1px solid #2a2a2a;
  border-radius: 6px;
  padding: 9px;
  cursor: pointer;
  font-size: 13px;
  font-family: "Inter", sans-serif;
}
.dlg-cancel:hover {
  background: #222;
  border-color: #333;
}
.dlg-delete {
  flex: 2;
  background: rgba(239, 68, 68, 0.15);
  color: #ef4444;
  border: 1px solid rgba(239, 68, 68, 0.3);
  border-radius: 6px;
  padding: 9px;
  cursor: pointer;
  font-size: 13px;
  font-weight: 500;
  font-family: "Inter", sans-serif;
}
.dlg-delete:hover {
  background: rgba(239, 68, 68, 0.28);
}
.dlg-delete.is-armed {
  background: #ef4444;
  color: #fff;
}
</style>
