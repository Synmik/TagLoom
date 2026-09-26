<template>
  <ModalShell title="Tag Manager" width="80vw" @close="requestClose">
    <div class="tm">
      <div class="tm-toolbar">
        <div class="tm-search">
          <Search :size="13" />
          <input
            ref="searchEl"
            v-model="query"
            class="tm-input"
            placeholder="Filter by name or alias"
            @keydown="onKeydown"
          />
          <button v-if="query" class="tm-icon-btn" title="Clear" @click="query = ''">
            <X :size="12" />
          </button>
        </div>
        <button class="tm-btn" @click="startCreate(false)"><Plus :size="13" /> New tag</button>
        <button class="tm-btn" @click="startCreate(true)">
          <FolderTree :size="13" /> New category
        </button>
        <select v-model="filterMode" class="tm-select" title="Show only these tags">
          <option value="all">All</option>
          <option value="unused">Unused (0 files)</option>
          <option value="categories">Categories</option>
          <option value="uncolored">Uncolored</option>
          <option value="no-parent">No parent</option>
        </select>
        <select v-model="sortMode" class="tm-select" title="Order of each level">
          <option value="tree">Sort: tree order</option>
          <option value="name">Sort: name</option>
          <option value="files">Sort: file count</option>
          <option value="created">Sort: created</option>
        </select>
        <span class="tm-flex"></span>
        <span class="tm-summary">{{ summary }}</span>
      </div>

      <div class="tm-main">
        <div
          ref="listEl"
          class="tm-list"
          tabindex="0"
          @scroll="onScroll"
          @keydown="onKeydown"
          @dragover.prevent
          @drop.prevent="onDrop"
        >
          <div class="tm-list-inner" :style="{ height: `${rows.length * ROW_H}px` }">
            <div
              v-for="(row, i) in visibleRows"
              :key="row.tag.id"
              class="tm-slot"
              :data-tag-id="row.tag.id"
              :style="{ top: `${(startIndex + i) * ROW_H}px`, height: `${ROW_H}px` }"
              draggable="true"
              @dragstart="onDragStart(row, $event)"
              @dragend="endInternalDrag"
            >
              <TagManagerRow
                :tag="row.tag"
                :depth="row.depth"
                :expanded="expanded.has(row.tag.id)"
                :has-children="row.hasChildren"
                :selected="isSelected(row.tag.id)"
                :cursor="cursorId === row.tag.id"
                :dim="row.dim"
                :unused="isUnused(row.tag)"
                :count="
                  row.tag.is_category === 1 ? aggregateCount(row.tag.id) : directCount(row.tag.id)
                "
                :aliases="aliasMap[row.tag.id] ?? []"
                :can-move-up="canMoveUp(row.tag)"
                :can-move-down="canMoveDown(row.tag)"
                @select="onSelect(row, $event)"
                @open="showFiles(row.tag)"
                @toggle="toggleExpand(row.tag.id)"
                @edit="startEdit(row.tag)"
                @move-up="move(row, -1)"
                @move-down="move(row, 1)"
                @delete="openDelete([row.tag])"
              />
            </div>
          </div>
          <div v-if="rows.length === 0" class="tm-empty">{{ emptyMessage }}</div>
        </div>

        <aside class="tm-detail">
          <template v-if="createMode">
            <div class="tm-detail-head">
              <span>{{ createMode.isCategory ? "New category" : "New tag" }}</span>
              <button class="tm-icon-btn" title="Cancel" @click="createMode = null">
                <X :size="13" />
              </button>
            </div>
            <TagDetailForm
              ref="createFormRef"
              :preset-parent-id="createParentId"
              :preset-category="createMode.isCategory"
              class="tm-detail-form"
              @saved="onSaved"
            />
          </template>

          <template v-else-if="singleTag">
            <div class="tm-detail-head">
              <span>Edit tag</span>
              <button class="tm-link" @click="showFiles(singleTag)">
                <CornerUpRight :size="12" />
                Show {{ aggregateCount(singleTag.id) }} file(s)
              </button>
            </div>
            <TagDetailForm
              :key="singleTag.id"
              ref="formRef"
              :tag="singleTag"
              :show-delete="false"
              class="tm-detail-form"
              @saved="onSaved"
              @show-files="showSelected"
              @merge-into="mergeIntoInstead"
            />
            <div class="tm-detail-actions">
              <button
                class="tm-btn"
                :disabled="!canMerge"
                @click="openMergeWith([singleTag], null)"
              >
                <Merge :size="13" /> Merge into…
              </button>
              <button class="tm-btn is-danger" @click="openDelete([singleTag])">
                <Trash2 :size="13" /> Delete…
              </button>
            </div>
          </template>

          <template v-else-if="selectedTags.length > 1">
            <div class="tm-detail-head">
              <span>{{ selectedTags.length }} tags selected</span>
            </div>
            <ul class="tm-selected-list">
              <li v-for="tag in selectedTags" :key="tag.id">{{ tag.name }}</li>
            </ul>
            <div class="tm-detail-note">
              {{ selectionFiles }} file link(s). Ctrl-click toggles, Shift-click selects a range.
            </div>
            <div class="tm-detail-actions">
              <button
                class="tm-btn"
                :disabled="!canMerge"
                @click="openMergeWith(selectedTags, null)"
              >
                <Merge :size="13" /> {{ mergeLabel }}
              </button>
              <button class="tm-btn is-danger" @click="openDelete(selectedTags)">
                <Trash2 :size="13" /> Delete…
              </button>
            </div>
            <div v-if="!kindsUniform" class="tm-detail-warn">
              Selection mixes categories and tags — merge only works within one kind.
            </div>
          </template>

          <div v-else class="tm-detail-empty">
            <p>Select a tag to edit it, or select several to merge them.</p>
            <button class="tm-btn is-primary" @click="startCreate(false)">＋ New tag</button>
            <p class="tm-detail-hint">
              Drag a row onto another to re-parent it, or use the arrows to reorder.<br />
              Search filters by name and alias · Ctrl+F focuses it · double click filters the
              gallery
            </p>
          </div>
        </aside>
      </div>

      <div class="tm-footer">
        <span class="tm-hint">
          ↑↓ move · Enter show files · Del delete · Ctrl+A select shown · Esc close
        </span>
        <span class="tm-flex"></span>
        <button v-if="selectedIds.length > 0" class="tm-btn" @click="selectedIds = []">
          Deselect
        </button>
        <button
          class="tm-btn"
          :disabled="!canMerge || busy"
          @click="openMergeWith(actionTargets, null)"
        >
          <Merge :size="13" /> {{ mergeLabel }}
        </button>
        <button
          class="tm-btn is-danger"
          :disabled="!hasTarget || busy"
          @click="openDelete(actionTargets)"
        >
          <Trash2 :size="13" /> Delete…
        </button>
      </div>
    </div>

    <MergeDialog
      v-if="mergeOpen && mergeSources.length > 0"
      :sources="mergeSources"
      :suggested-target-id="suggestedTargetId"
      @cancel="closeMerge"
      @confirm="runMerge"
    />
    <DeleteTagDialog
      v-if="deleteTargets.length > 0"
      :tags="deleteTargets"
      @cancel="deleteTargets = []"
      @confirm="runDelete"
      @merge-instead="mergeInsteadOfDelete"
    />
    <ConfirmDialog
      v-if="discardOpen"
      title="Unsaved changes"
      message="The tag you were editing has unsaved changes. Discard them?"
      confirm-text="Discard"
      @confirm="onDiscardConfirm"
      @cancel="onDiscardCancel"
    />
  </ModalShell>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from "vue";
import { CornerUpRight, FolderTree, Merge, Plus, Search, Trash2, X } from "@lucide/vue";
import ModalShell from "../../common/ModalShell.vue";
import ConfirmDialog from "../../common/ConfirmDialog.vue";
import TagDetailForm from "./TagDetailForm.vue";
import TagManagerRow from "./TagManagerRow.vue";
import MergeDialog from "./MergeDialog.vue";
import DeleteTagDialog from "./DeleteTagDialog.vue";
import { useTagsStore } from "../../../stores/tags";
import { useFiltersStore } from "../../../stores/filters";
import { useFilesStore } from "../../../stores/files";
import { useToast } from "../../../composables/useToast";
import { beginInternalDrag, endInternalDrag } from "../../../composables/useDragDrop";
import type { ChildStrategy, Tag, TagAlias } from "../../../types/tag";

const emit = defineEmits<{ close: [] }>();

/** Fixed row height — the list is virtualised, so rows must not change height. */
const ROW_H = 30;

const tagsStore = useTagsStore();
const filtersStore = useFiltersStore();
const filesStore = useFilesStore();
const { success, error: toastError } = useToast();

/* ── state ─────────────────────────────────────────────────────────── */

const query = ref("");
const filterMode = ref<FilterMode>("all");
const sortMode = ref<"tree" | "name" | "files" | "created">("tree");
const selectedIds = ref<number[]>([]);
const cursorId = ref<number | null>(null);
const expanded = ref<Set<number>>(new Set());
const aliasMap = ref<Record<number, string[]>>({});
const createMode = ref<{ isCategory: boolean } | null>(null);
const mergeOpen = ref(false);
const deleteTargets = ref<Tag[]>([]);
const discardOpen = ref(false);
const busy = ref(false);

const searchEl = ref<HTMLInputElement | null>(null);
const listEl = ref<HTMLElement | null>(null);
const formRef = ref<InstanceType<typeof TagDetailForm> | null>(null);
const createFormRef = ref<InstanceType<typeof TagDetailForm> | null>(null);

const scrollTop = ref(0);
const viewHeight = ref(480);
let resizeObserver: ResizeObserver | null = null;
/** Action to run once unsaved edits are discarded. */
let pendingAction: (() => void) | null = null;

/* ── hierarchy, derived once per tag change ───────────────────────── */

const byId = computed(() => new Map(tagsStore.tags.map((t) => [t.id, t] as const)));

/**
 * Children per parent, in display order. A tag whose parent is gone is treated
 * as a root, otherwise it would be unreachable — and undeletable — in this UI.
 */
/** Sibling order. "Tree order" is the stored sort_order; the others are computed,
 *  so the hierarchy is kept while a level is re-sorted. */
function siblingOrder(a: Tag, b: Tag): number {
  const byName = a.name.localeCompare(b.name);
  switch (sortMode.value) {
    case "name":
      return byName;
    case "files":
      return usageCount(b) - usageCount(a) || byName;
    case "created":
      return (a.created_at || "").localeCompare(b.created_at || "") || byName;
    default:
      return a.sort_order - b.sort_order || byName;
  }
}

const childrenByParent = computed(() => {
  const map = new Map<number | null, Tag[]>();
  const ordered = [...tagsStore.tags].sort(siblingOrder);
  for (const tag of ordered) {
    const parentId = tag.parent_id != null && byId.value.has(tag.parent_id) ? tag.parent_id : null;
    const siblings = map.get(parentId);
    if (siblings) siblings.push(tag);
    else map.set(parentId, [tag]);
  }
  return map;
});

const searchNeedle = computed(() => query.value.trim().toLowerCase());

/** The filter menu. Independent of the search box; both must accept a tag. */
type FilterMode = "all" | "unused" | "categories" | "uncolored" | "no-parent";

function passesFilter(tag: Tag): boolean {
  switch (filterMode.value) {
    case "unused":
      return usageCount(tag) === 0;
    case "categories":
      return tag.is_category === 1;
    case "uncolored":
      return !tag.color;
    case "no-parent":
      return tag.parent_id == null || !byId.value.has(tag.parent_id);
    default:
      return true;
  }
}

/** Tags the search box and the filter menu accept. Null when neither is applied,
 *  which is the only case where collapsed levels stay collapsed. */
const matchedIds = computed<Set<number> | null>(() => {
  const needle = searchNeedle.value;
  const filtering = filterMode.value !== "all";
  if (!needle && !filtering) return null;
  const hits = new Set<number>();
  for (const tag of tagsStore.tags) {
    if (!passesFilter(tag)) continue;
    if (!needle) {
      hits.add(tag.id);
      continue;
    }
    if (tag.name.toLowerCase().includes(needle)) {
      hits.add(tag.id);
      continue;
    }
    if ((aliasMap.value[tag.id] ?? []).some((alias) => alias.toLowerCase().includes(needle))) {
      hits.add(tag.id);
    }
  }
  return hits;
});

/** A match is only useful with its ancestry, and a parent without its children is misleading. */
function collectRelatives(ids: Set<number>): Set<number> {
  const shown = new Set<number>(ids);
  for (const id of ids) {
    let current = byId.value.get(id);
    while (current?.parent_id != null) {
      const parent = byId.value.get(current.parent_id);
      if (!parent || shown.has(parent.id)) break; // already shown, or a cycle
      shown.add(parent.id);
      current = parent;
    }
    const stack = [...(childrenByParent.value.get(id) ?? [])];
    while (stack.length > 0) {
      const child = stack.pop();
      if (!child || shown.has(child.id)) continue;
      shown.add(child.id);
      stack.push(...(childrenByParent.value.get(child.id) ?? []));
    }
  }
  return shown;
}

interface ManagerRow {
  tag: Tag;
  depth: number;
  /** Shown only as context around a search hit. */
  dim: boolean;
  hasChildren: boolean;
}

const rows = computed<ManagerRow[]>(() => {
  const matched = matchedIds.value;
  const shown = matched ? collectRelatives(matched) : null;
  const out: ManagerRow[] = [];
  // A corrupt parent chain must not loop forever.
  const seen = new Set<number>();

  const walk = (parentId: number | null, depth: number) => {
    for (const tag of childrenByParent.value.get(parentId) ?? []) {
      if (seen.has(tag.id) || depth > 32) continue;
      if (shown && !shown.has(tag.id)) continue;
      seen.add(tag.id);
      const children = childrenByParent.value.get(tag.id) ?? [];
      out.push({
        tag,
        depth,
        dim: matched !== null && !matched.has(tag.id),
        hasChildren: children.length > 0,
      });
      // Searching expands the tree so a hit is never hidden behind a collapse.
      if (children.length > 0 && (shown !== null || expanded.value.has(tag.id))) {
        walk(tag.id, depth + 1);
      }
    }
  };
  walk(null, 0);
  return out;
});

/* ── virtual window ───────────────────────────────────────────────── */

const startIndex = computed(() => Math.max(0, Math.floor(scrollTop.value / ROW_H) - 6));
const visibleRows = computed(() =>
  rows.value.slice(startIndex.value, startIndex.value + Math.ceil(viewHeight.value / ROW_H) + 12),
);

function onScroll(event: Event) {
  scrollTop.value = (event.target as HTMLElement).scrollTop;
}

function scrollRowIntoView(tagId: number) {
  const index = rows.value.findIndex((row) => row.tag.id === tagId);
  if (index < 0) return;
  const top = index * ROW_H;
  const bottom = top + ROW_H;
  const el = listEl.value;
  if (!el) return;
  if (top < el.scrollTop) el.scrollTop = top;
  else if (bottom > el.scrollTop + el.clientHeight) el.scrollTop = bottom - el.clientHeight;
}

/* ── counts and aliases ───────────────────────────────────────────── */

function directCount(tagId: number): number {
  return tagsStore.tagCounts[tagId] ?? 0;
}
function aggregateCount(tagId: number): number {
  return tagsStore.getAggregateCount(tagId);
}

function groupAliases(rows: TagAlias[]): Record<number, string[]> {
  const map: Record<number, string[]> = {};
  for (const row of rows) {
    (map[row.tag_id] ??= []).push(row.alias);
  }
  return map;
}

async function refreshData(): Promise<void> {
  await tagsStore.loadTags();
  aliasMap.value = groupAliases(await tagsStore.loadAllTagAliases());
  const alive = new Set(tagsStore.tags.map((t) => t.id));
  selectedIds.value = selectedIds.value.filter((id) => alive.has(id));
  if (cursorId.value !== null && !alive.has(cursorId.value)) cursorId.value = null;
}

const categories = computed(() => tagsStore.tags.filter((t) => t.is_category === 1).length);

const filterEmptyLabels: Record<Exclude<FilterMode, "all">, string> = {
  unused: "unused tags",
  categories: "categories",
  uncolored: "uncolored tags",
  "no-parent": "tags without a parent",
};

/** Why the list is empty, rather than a bare "no results". */
const emptyMessage = computed(() => {
  if (tagsStore.tags.length === 0) return "No tags yet — create one to get started";
  const reasons: string[] = [];
  if (searchNeedle.value) reasons.push(`match for "${query.value.trim()}"`);
  if (filterMode.value !== "all") reasons.push(filterEmptyLabels[filterMode.value]);
  return reasons.length > 0 ? `Nothing to show — no ${reasons.join(" and ")}` : "Nothing to show";
});
const summary = computed(() => {
  const total = tagsStore.tags.length;
  if (matchedIds.value === null) return `${total} tags · ${categories.value} categories`;
  // Context rows (ancestors, descendants) are not matches, so they do not count.
  const matches = rows.value.reduce((n, row) => n + (row.dim ? 0 : 1), 0);
  return `filtered ${matches} / ${total} tags`;
});

/* ── selection ────────────────────────────────────────────────────── */

const selectedTags = computed(() =>
  selectedIds.value.map((id) => byId.value.get(id)).filter((t): t is Tag => !!t),
);
const singleTag = computed<Tag | null>(() =>
  selectedIds.value.length === 1 ? (byId.value.get(selectedIds.value[0]) ?? null) : null,
);
const cursorTag = computed<Tag | null>(() =>
  cursorId.value === null ? null : (byId.value.get(cursorId.value) ?? null),
);
const selectionFiles = computed(() =>
  selectedTags.value.reduce((sum, tag) => sum + directCount(tag.id), 0),
);
/** Categories and plain tags never merge, so a mixed selection cannot. */
const kindsUniform = computed(() => {
  if (selectedTags.value.length === 0) return true;
  const kind = selectedTags.value[0].is_category;
  return selectedTags.value.every((t) => t.is_category === kind);
});
const canMerge = computed(() => selectedIds.value.length >= 1 && kindsUniform.value && !busy.value);
const hasTarget = computed(() => selectedIds.value.length > 0 || cursorTag.value !== null);
const actionTargets = computed<Tag[]>(() =>
  selectedTags.value.length > 0 ? selectedTags.value : cursorTag.value ? [cursorTag.value] : [],
);
/**
 * Set when a merge is opened from somewhere other than the selection — the
 * duplicate-name offer in the form, or "merge instead" in the delete dialog.
 * Sources are fixed then, and a target may be suggested.
 */
const mergeFrom = ref<{ sources: Tag[]; targetId: number | null } | null>(null);
/** Sources for the merge dialog: an explicit request, else the selection or focused row. */
const mergeSources = computed(() => mergeFrom.value?.sources ?? actionTargets.value);
const suggestedTargetId = computed(() => mergeFrom.value?.targetId ?? null);

function openMergeWith(sources: Tag[], targetId: number | null) {
  if (sources.length === 0) return;
  mergeFrom.value = { sources, targetId };
  mergeOpen.value = true;
}

function closeMerge() {
  mergeOpen.value = false;
  mergeFrom.value = null;
}

/** Count used for dimming and sorting: links of a category are its children's. */
function usageCount(tag: Tag): number {
  return tag.is_category === 1 ? aggregateCount(tag.id) : directCount(tag.id);
}

function isUnused(tag: Tag): boolean {
  return tag.is_category !== 1 && directCount(tag.id) === 0;
}

function isSelected(tagId: number): boolean {
  return selectedIds.value.includes(tagId);
}

/** `isDirty` is an exposed computed — unwrapped on the public instance. */
function formIsDirty(): boolean {
  for (const form of [formRef.value, createFormRef.value]) {
    const flag = (form as unknown as { isDirty?: boolean | { value: boolean } } | null)?.isDirty;
    if (typeof flag === "object" && flag !== null) {
      if (flag.value) return true;
    } else if (flag === true) {
      return true;
    }
  }
  return false;
}

/** Run `action` now, or after the user accepts losing unsaved edits. */
function guardUnsaved(action: () => void): void {
  if (formIsDirty()) {
    pendingAction = action;
    discardOpen.value = true;
    return;
  }
  action();
}

function onDiscardConfirm() {
  discardOpen.value = false;
  const action = pendingAction;
  pendingAction = null;
  action?.();
}

function onDiscardCancel() {
  discardOpen.value = false;
  pendingAction = null;
}

function requestClose() {
  guardUnsaved(() => emit("close"));
}

/** Selecting a tag always leaves the "New tag" pane — otherwise an empty create
 *  form stays on screen over a selected tag, with no name to save. */
function commitSelection(ids: number[]) {
  createMode.value = null;
  selectedIds.value = ids;
}

function onSelect(row: ManagerRow, event: MouseEvent) {
  const id = row.tag.id;
  const at = rows.value.findIndex((r) => r.tag.id === id);
  cursorId.value = id;

  if (event.shiftKey && anchorIndex.value !== null) {
    // Shift extends the last plain click, matching the file list behaviour.
    const from = Math.min(anchorIndex.value, at);
    const to = Math.max(anchorIndex.value, at);
    guardUnsaved(() => commitSelection(rows.value.slice(from, to + 1).map((r) => r.tag.id)));
  } else if (event.ctrlKey || event.metaKey) {
    const next = [...selectedIds.value];
    const existing = next.indexOf(id);
    if (existing >= 0) next.splice(existing, 1);
    else next.push(id);
    anchorIndex.value = at;
    guardUnsaved(() => commitSelection(next));
  } else {
    anchorIndex.value = at;
    guardUnsaved(() => {
      createMode.value = null;
      commitSelection([id]);
    });
  }
  focusList();
}

/** Row index a shift-range extends from. */
const anchorIndex = ref<number | null>(null);

function toggleExpand(tagId: number) {
  const next = new Set(expanded.value);
  if (next.has(tagId)) next.delete(tagId);
  else next.add(tagId);
  expanded.value = next;
}

/* ── keyboard ─────────────────────────────────────────────────────── */

function focusList() {
  void nextTick(() => listEl.value?.focus());
}

function moveCursorTo(index: number) {
  const bounded = Math.max(0, Math.min(rows.value.length - 1, index));
  const row = rows.value[bounded];
  if (!row) return;
  cursorId.value = row.tag.id;
  guardUnsaved(() => commitSelection([row.tag.id]));
  scrollRowIntoView(row.tag.id);
}

/** A dialog on top of the manager owns the keyboard while it is open — its own
 *  window-level Escape must not be swallowed by the list handler below. */
const dialogOpen = computed(
  () => mergeOpen.value || deleteTargets.value.length > 0 || discardOpen.value,
);

function onKeydown(event: KeyboardEvent) {
  if (dialogOpen.value) return;

  // Ctrl+F belongs to the manager's own filter box, not the gallery search.
  if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === "f") {
    event.preventDefault();
    event.stopPropagation();
    searchEl.value?.focus();
    searchEl.value?.select();
    return;
  }

  const fromSearch = event.target === searchEl.value;
  if (fromSearch) {
    if (event.key === "Escape") {
      event.stopPropagation();
      if (query.value) query.value = "";
      else requestClose();
      return;
    }
    if (event.key === "ArrowDown" && rows.value.length > 0) {
      event.preventDefault();
      event.stopPropagation();
      focusList();
      moveCursorTo(0);
      return;
    }
    if (event.key === "Enter" && rows.value.length > 0) {
      event.preventDefault();
      event.stopPropagation();
      const first = rows.value[0];
      commitSelection([first.tag.id]);
      showFiles(first.tag);
    }
    return;
  }

  const index = rows.value.findIndex((row) => row.tag.id === cursorId.value);
  switch (event.key) {
    case "ArrowDown":
      event.preventDefault();
      event.stopPropagation();
      moveCursorTo(index < 0 ? 0 : index + 1);
      return;
    case "ArrowUp":
      if (index <= 0 && listIsScrolledToTop()) {
        searchEl.value?.focus();
        return;
      }
      event.preventDefault();
      event.stopPropagation();
      moveCursorTo(index < 0 ? 0 : index - 1);
      return;
    case "ArrowRight": {
      const row = rows.value[index];
      if (!row) return;
      event.preventDefault();
      event.stopPropagation();
      if (row.hasChildren && !expanded.value.has(row.tag.id)) toggleExpand(row.tag.id);
      else if (row.hasChildren) moveCursorTo(index + 1);
      return;
    }
    case "ArrowLeft": {
      const row = rows.value[index];
      if (!row) return;
      event.preventDefault();
      event.stopPropagation();
      if (row.hasChildren && expanded.value.has(row.tag.id)) toggleExpand(row.tag.id);
      else if (row.tag.parent_id != null) {
        cursorId.value = row.tag.parent_id;
        guardUnsaved(() => commitSelection([row.tag.parent_id as number]));
        scrollRowIntoView(row.tag.parent_id);
      }
      return;
    }
    case "Enter":
      if (!cursorTag.value) return;
      event.preventDefault();
      event.stopPropagation();
      showFiles(cursorTag.value);
      return;
    case "Delete":
    case "Backspace":
      if (!hasTarget.value) return;
      event.preventDefault();
      event.stopPropagation();
      openDelete(actionTargets.value);
      return;
    case "a":
    case "A":
      if (!event.ctrlKey && !event.metaKey) return;
      event.preventDefault();
      event.stopPropagation();
      commitSelection(rows.value.map((row) => row.tag.id));
      return;
    case "Escape":
      // Layered: drop the selection, then the filter text, then close.
      event.preventDefault();
      event.stopPropagation();
      if (selectedIds.value.length > 0) selectedIds.value = [];
      else if (query.value) query.value = "";
      else requestClose();
      return;
  }
}

function listIsScrolledToTop(): boolean {
  return (listEl.value?.scrollTop ?? 0) <= 0;
}

/* ── actions ──────────────────────────────────────────────────────── */

function showFiles(tag: Tag) {
  const tags = selectedTags.value.length > 1 ? selectedTags.value : [tag];
  // One group means "any of these"; separate groups would mean "all of these".
  const group = tags.flatMap((t) => [t.id, ...tagsStore.getAllDescendantIds(t.id)]);
  filtersStore.setTagFilter([group]);
  void filesStore.reloadFiles();
  emit("close");
}

function startCreate(isCategory: boolean) {
  guardUnsaved(() => {
    commitSelection([]);
    createMode.value = { isCategory };
  });
}

const createParentId = computed<number | null>(() => {
  const cursor = cursorTag.value;
  if (!cursor) return null;
  return cursor.is_category === 1 ? cursor.id : (cursor.parent_id ?? null);
});

async function onSaved() {
  createMode.value = null;
  busy.value = true;
  try {
    await refreshData();
  } finally {
    busy.value = false;
  }
}

function openDelete(targets: Tag[]) {
  if (targets.length === 0) return;
  deleteTargets.value = targets;
}

/** "Merge into…", or "Merge 4 into…" once a batch is selected. */
const mergeLabel = computed(() =>
  selectedIds.value.length >= 2 ? `Merge ${selectedIds.value.length} into…` : "Merge into…",
);

/** The detail pane's "Show N files" link, and the form's warning link. */
function showSelected() {
  if (singleTag.value) showFiles(singleTag.value);
}

/**
 * The form found a tag with the name being typed; offer to fold the edited tag
 * into that one instead of renaming onto it.
 */
function mergeIntoInstead(targetId: number) {
  if (singleTag.value) openMergeWith([singleTag.value], targetId);
}

/** Deleting a used tag — merging keeps its file links, so offer it. */
function mergeInsteadOfDelete() {
  const sources = deleteTargets.value;
  deleteTargets.value = [];
  if (sources.length > 0) openMergeWith(sources, null);
}

async function runDelete(strategy: ChildStrategy) {
  const targets = deleteTargets.value;
  busy.value = true;
  try {
    await tagsStore.deleteTagsSafely(
      targets.map((t) => t.id),
      strategy,
    );
    deleteTargets.value = [];
    selectedIds.value = [];
    await refreshData();
    success(`Deleted ${targets.length} ${targets.length === 1 ? "tag" : "tags"}`);
  } catch (e) {
    toastError(`Delete failed: ${errorText(e)}`);
  } finally {
    busy.value = false;
  }
}

async function runMerge(targetId: number) {
  const sources = mergeSources.value;
  const target = byId.value.get(targetId);
  busy.value = true;
  try {
    const result = await tagsStore.mergeTags(
      sources.map((t) => t.id),
      targetId,
    );
    closeMerge();
    // Keep the manager open on the tag that survived, per the merge flow.
    selectedIds.value = [targetId];
    cursorId.value = targetId;
    await refreshData();
    success(mergeMessage(sources.length, target?.name ?? "", result));
  } catch (e) {
    toastError(`Merge failed: ${errorText(e)}`);
  } finally {
    busy.value = false;
  }
}

function mergeMessage(
  sourceCount: number,
  targetName: string,
  result: Awaited<ReturnType<typeof tagsStore.mergeTags>>,
): string {
  if (!result) return `Merged ${sourceCount} tag(s) into "${targetName}"`;
  const parts = [
    `${result.files_moved} file link(s) moved`,
    result.duplicates > 0 ? `${result.duplicates} already tagged` : "",
    result.aliases_moved + result.names_adopted > 0
      ? `${result.aliases_moved + result.names_adopted} alias(es) kept`
      : "",
    result.children_moved > 0 ? `${result.children_moved} child tag(s) moved` : "",
  ].filter(Boolean);
  return `Merged ${sourceCount} tag(s) into "${targetName}" — ${parts.join(", ")}`;
}

function errorText(e: unknown): string {
  if (e instanceof Error) return e.message;
  return String(e);
}

/* ── drag to re-parent ────────────────────────────────────────────── */

function onDragStart(row: ManagerRow, event: DragEvent) {
  cursorId.value = row.tag.id;
  // Keep the app-level "Drop files to import" overlay away: Wails flags the root
  // drop target for any drag over the window, this one included.
  beginInternalDrag();
  if (event.dataTransfer) {
    // A custom type, never "Files", so the app-level import overlay stays away:
    // useDragDrop only reacts to drops that carry files.
    event.dataTransfer.setData("application/x-tagloom-tag", String(row.tag.id));
    event.dataTransfer.effectAllowed = "move";
  }
}

/**
 * Dropping a row onto another re-parents it onto that row. Reordering within a
 * level uses the move buttons — that is also the fallback when dragging is
 * awkward on a long list.
 */
async function onDrop(event: DragEvent) {
  const raw = event.dataTransfer?.getData("application/x-tagloom-tag");
  if (!raw) return; // a drop of files, or of anything else — not ours
  const draggedId = Number(raw);
  const slot = (event.target as HTMLElement | null)?.closest(".tm-slot");
  const targetId = Number(slot?.getAttribute("data-tag-id") ?? "-1");
  const target = targetId >= 0 ? byId.value.get(targetId) : undefined;
  if (!target || target.id === draggedId) return;

  try {
    await tagsStore.moveTag(draggedId, target.id, 0);
    await refreshData();
    if (!expanded.value.has(target.id)) toggleExpand(target.id);
  } catch (e) {
    toastError(`Move failed: ${errorText(e)}`);
  }
}

/* ── move up / down within a level ────────────────────────────────── */

function siblingsOf(tag: Tag): Tag[] {
  const parentId = tag.parent_id != null && byId.value.has(tag.parent_id) ? tag.parent_id : null;
  return childrenByParent.value.get(parentId) ?? [];
}

function canMoveUp(tag: Tag): boolean {
  const siblings = siblingsOf(tag);
  return siblings.length > 1 && siblings[0].id !== tag.id;
}

function canMoveDown(tag: Tag): boolean {
  const siblings = siblingsOf(tag);
  return siblings.length > 1 && siblings[siblings.length - 1].id !== tag.id;
}

/** Row pencil: select the tag so the form renders, then focus its name field. */
function startEdit(tag: Tag) {
  selectedIds.value = [tag.id];
  cursorId.value = tag.id;
  void nextTick(() => formRef.value?.focusName());
}

async function move(row: ManagerRow, direction: -1 | 1) {
  const siblings = siblingsOf(row.tag);
  const index = siblings.findIndex((sibling) => sibling.id === row.tag.id);
  const target = siblings[index + direction];
  if (!target) return;
  // Moving up means landing before the previous sibling; moving down means
  // landing after it, i.e. before the sibling that followed it.
  const beforeId = direction < 0 ? target.id : (siblings[index + 2]?.id ?? 0);
  const parentId =
    row.tag.parent_id != null && byId.value.has(row.tag.parent_id) ? row.tag.parent_id : null;
  try {
    await tagsStore.moveTag(row.tag.id, parentId, beforeId);
    await refreshData();
  } catch (e) {
    toastError(`Move failed: ${errorText(e)}`);
  }
}

/* ── lifecycle ────────────────────────────────────────────────────── */

onMounted(async () => {
  resizeObserver = new ResizeObserver(() => {
    viewHeight.value = listEl.value?.clientHeight ?? 480;
  });
  if (listEl.value) {
    viewHeight.value = listEl.value.clientHeight;
    resizeObserver.observe(listEl.value);
  }
  await refreshData();
  // Categories open, plain tags closed: the top level stays readable at scale.
  if (expanded.value.size === 0) {
    expanded.value = new Set(tagsStore.tags.filter((t) => t.is_category === 1).map((t) => t.id));
  }
  focusList();
});

onBeforeUnmount(() => resizeObserver?.disconnect());

// Tags changed elsewhere in the app (batch edit, right panel) — resync without
// asking the backend again, the store already holds the new list.
watch(
  () => tagsStore.tags.length,
  () => {
    const alive = new Set(tagsStore.tags.map((t) => t.id));
    selectedIds.value = selectedIds.value.filter((id) => alive.has(id));
  },
);
</script>

<style scoped>
/* ModalShell owns the box; the manager fills it with a three-part layout. */
:deep(.modal-body) {
  padding: 0;
  gap: 0;
  overflow: hidden;
}
:deep(.modal) {
  min-width: 860px;
  height: 85vh;
  max-height: 85vh;
}

.tm {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
}
.tm-toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 14px;
  border-bottom: 1px solid #1a1a1a;
  flex-shrink: 0;
}
.tm-search {
  display: flex;
  align-items: center;
  gap: 6px;
  background: #1a1a1a;
  border: 1px solid #2a2a2a;
  border-radius: 6px;
  padding: 0 8px;
  height: 28px;
  width: 240px;
  color: #666;
}
.tm-search:focus-within {
  border-color: #3a3a3a;
}
.tm-input {
  background: none;
  border: none;
  outline: none;
  color: #e8e8e8;
  font-size: 12px;
  font-family: "Inter", sans-serif;
  width: 100%;
}
.tm-icon-btn {
  background: none;
  border: none;
  color: #666;
  cursor: pointer;
  display: flex;
  align-items: center;
  padding: 2px;
  border-radius: 3px;
}
.tm-icon-btn:hover {
  color: #e8e8e8;
  background: #2a2a2a;
}
.tm-btn {
  display: flex;
  align-items: center;
  gap: 5px;
  background: #1a1a1a;
  border: 1px solid #2a2a2a;
  border-radius: 6px;
  color: #ccc;
  padding: 5px 10px;
  font-size: 12px;
  font-family: "Inter", sans-serif;
  cursor: pointer;
  white-space: nowrap;
  transition:
    background 0.15s,
    border-color 0.15s;
}
.tm-btn:hover:not(:disabled) {
  background: #232323;
  border-color: #333;
}
.tm-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}
.tm-btn.is-danger {
  color: #ef4444;
  border-color: rgba(239, 68, 68, 0.2);
  background: rgba(239, 68, 68, 0.08);
}
.tm-btn.is-danger:hover:not(:disabled) {
  background: rgba(239, 68, 68, 0.18);
}
.tm-summary {
  font-size: 11px;
  color: #666;
  font-family: "Inter", sans-serif;
}
.tm-select {
  background: #1a1a1a;
  border: 1px solid #2a2a2a;
  border-radius: 6px;
  color: #ccc;
  padding: 5px 6px;
  font-size: 11px;
  font-family: "Inter", sans-serif;
  outline: none;
  cursor: pointer;
}
.tm-select:hover {
  border-color: #333;
}
.tm-select option {
  background: #1a1a1a;
  color: #e8e8e8;
}
.tm-flex {
  flex: 1;
}

.tm-main {
  display: flex;
  flex: 1;
  min-height: 0;
}
.tm-list {
  flex: 1;
  min-width: 0;
  overflow-y: auto;
  position: relative;
  outline: none;
  padding: 4px 0;
}
.tm-list-inner {
  position: relative;
}
.tm-slot {
  position: absolute;
  left: 0;
  right: 0;
}
.tm-empty {
  padding: 24px;
  text-align: center;
  color: #666;
  font-size: 12px;
  font-family: "Inter", sans-serif;
}

.tm-detail {
  width: 340px;
  flex-shrink: 0;
  border-left: 1px solid #1a1a1a;
  padding: 12px 14px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.tm-detail-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  color: #666;
  font-family: "Inter", sans-serif;
}
.tm-link {
  display: flex;
  align-items: center;
  gap: 4px;
  background: none;
  border: none;
  color: #4ade80;
  font-size: 11px;
  font-family: "Inter", sans-serif;
  cursor: pointer;
  text-transform: none;
  letter-spacing: 0;
  padding: 2px 4px;
  border-radius: 4px;
}
.tm-link:hover {
  background: rgba(34, 197, 94, 0.1);
}
.tm-detail-form :deep(.form-actions) {
  /* The footer owns Merge / Delete; the form keeps only Save. */
  padding-top: 4px;
}
.tm-detail-actions {
  display: flex;
  gap: 8px;
}
.tm-selected-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 3px;
  font-size: 12px;
  color: #ccc;
  font-family: "Inter", sans-serif;
  max-height: 220px;
  overflow-y: auto;
}
.tm-detail-note,
.tm-detail-warn {
  font-size: 11px;
  color: #666;
  font-family: "Inter", sans-serif;
  line-height: 1.5;
}
.tm-detail-warn {
  color: #f59e0b;
}
.tm-detail-empty {
  color: #888;
  font-size: 12px;
  font-family: "Inter", sans-serif;
  line-height: 1.6;
  padding-top: 8px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
}
.tm-detail-empty p {
  margin: 0;
}
.tm-detail-hint {
  color: #555;
  font-size: 10px;
  line-height: 1.7;
}

.tm-footer {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 14px;
  border-top: 1px solid #1a1a1a;
  flex-shrink: 0;
}
.tm-hint {
  font-size: 11px;
  color: #555;
  font-family: "Inter", sans-serif;
}
</style>
