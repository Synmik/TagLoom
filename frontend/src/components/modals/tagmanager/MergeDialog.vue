<template>
  <Teleport to="body">
    <div class="dlg-overlay" @click.self="$emit('cancel')">
      <div class="dlg">
        <div class="dlg-header">
          <h3>Merge {{ sources.length }} {{ kindLabel }} into…</h3>
          <button class="dlg-close" @click="$emit('cancel')"><X :size="16" /></button>
        </div>

        <div class="dlg-body">
          <div class="dlg-section">
            <div class="dlg-label">Merge these</div>
            <ul class="dlg-source-list">
              <li v-for="source in sources" :key="source.id">
                <span class="dlg-dot" :style="{ background: source.color || '#666' }"></span>
                <span class="dlg-source-name">{{ source.name }}</span>
                <span class="dlg-source-meta">{{ sourceMeta(source) }}</span>
              </li>
            </ul>
          </div>

          <div class="dlg-section">
            <div class="dlg-label">Into target</div>
            <input
              ref="searchInput"
              v-model="query"
              class="dlg-input"
              placeholder="Search target tags"
            />
            <div class="dlg-target-list">
              <button
                v-for="candidate in candidates"
                :key="candidate.id"
                class="dlg-target"
                :class="{ 'is-picked': targetId === candidate.id }"
                @click="pick(candidate.id)"
              >
                <span class="dlg-dot" :style="{ background: candidate.color || '#666' }"></span>
                <span class="dlg-target-name">{{ candidate.name }}</span>
                <span v-if="candidate.is_category === 1" class="dlg-badge">cat</span>
                <span class="dlg-flex"></span>
                <span class="dlg-count">{{ countOf(candidate.id) }}</span>
              </button>
              <div v-if="candidates.length === 0" class="dlg-empty">
                No other {{ kindLabel }}s to merge into
              </div>
            </div>
          </div>

          <div class="dlg-preview">
            <div>
              Moves up to <b>{{ totals.files }}</b> file link(s) to the target
            </div>
            <div>
              Keeps <b>{{ totals.aliases }}</b> alias(es), so saved searches still resolve
            </div>
            <div v-if="totals.children > 0">
              Adopts <b>{{ totals.children }}</b> child tag(s) under the target
            </div>
            <div class="dlg-note">
              Files that already carry the target are counted once. Exact numbers are reported after
              the merge.
            </div>
          </div>
        </div>

        <div class="dlg-footer">
          <button class="dlg-cancel" @click="$emit('cancel')">Cancel</button>
          <button
            class="dlg-confirm"
            :class="{ 'is-armed': confirming }"
            :disabled="targetId === null"
            @click="onConfirm"
          >
            <template v-if="targetId === null">Pick a target</template>
            <template v-else-if="confirming">Confirm merge into "{{ targetName }}"</template>
            <template v-else>Merge into "{{ targetName }}"</template>
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";
import { X } from "@lucide/vue";
import { useTagsStore } from "../../../stores/tags";
import type { Tag, TagUsage } from "../../../types/tag";

const props = withDefaults(
  defineProps<{
    /** Tags that will be folded into the target and then deleted. */
    sources: Tag[];
    /** Pre-picked target — e.g. the tag whose name was typed into the form. */
    suggestedTargetId?: number | null;
  }>(),
  { suggestedTargetId: null },
);

const emit = defineEmits<{
  cancel: [];
  confirm: [targetId: number];
}>();

const tagsStore = useTagsStore();

const usage = ref<Record<number, TagUsage>>({});
const query = ref("");
const targetId = ref<number | null>(null);
/** Two-step confirm: the destructive button must be pressed twice. */
const confirming = ref(false);
const searchInput = ref<HTMLInputElement | null>(null);

const kindLabel = computed(() =>
  props.sources.length > 0 && props.sources[0].is_category === 1 ? "category" : "tag",
);

/** Merges only happen within one kind — see `candidates`. */
const wantedKind = computed(() => (props.sources.length > 0 ? props.sources[0].is_category : 0));

// A merge only works within one kind: a category and a plain tag would produce
// rows the backend rejects (categories are not assignable to files).
const candidates = computed(() => {
  const sourceIds = new Set(props.sources.map((t) => t.id));
  const needle = query.value.trim().toLowerCase();
  return tagsStore.tags
    .filter((t) => !sourceIds.has(t.id) && t.is_category === wantedKind.value)
    .filter((t) => needle === "" || t.name.toLowerCase().includes(needle))
    .sort((a, b) => a.sort_order - b.sort_order || a.name.localeCompare(b.name));
});

/**
 * A suggested target is pre-picked, unless it is one of the sources or a
 * different kind — the backend refuses a merge across kinds, so pre-picking it
 * would only queue up an error.
 */
const usableSuggestion = computed(() => {
  const id = props.suggestedTargetId;
  if (id == null || props.sources.some((source) => source.id === id)) return null;
  const target = tagsStore.tags.find((t) => t.id === id);
  if (!target || target.is_category !== wantedKind.value) return null;
  return id;
});
watch(
  usableSuggestion,
  (id) => {
    if (id !== null) targetId.value = id;
  },
  { immediate: true },
);

const targetName = computed(() => tagsStore.tags.find((t) => t.id === targetId.value)?.name ?? "");

const totals = computed(() => {
  let files = 0;
  let aliases = 0;
  let children = 0;
  for (const source of props.sources) {
    const u = usage.value[source.id];
    if (!u) continue;
    files += u.direct_files;
    // Each source name is adopted as an alias of the target too.
    aliases += u.aliases + 1;
    children += u.children;
  }
  return { files, aliases, children };
});

function countOf(tagId: number): number {
  return tagsStore.tagCounts[tagId] ?? 0;
}

function sourceMeta(source: Tag): string {
  const u = usage.value[source.id];
  if (!u) return "";
  const parts = [`${u.direct_files} files`];
  if (u.aliases > 0) parts.push(`${u.aliases} aliases`);
  if (u.children > 0) parts.push(`${u.children} children`);
  return parts.join(" · ");
}

function pick(id: number) {
  targetId.value = id;
  confirming.value = false;
}

function onConfirm() {
  if (targetId.value === null) return;
  if (!confirming.value) {
    confirming.value = true;
    return;
  }
  emit("confirm", targetId.value);
}

// Escape closes this dialog only — captured on window so the app-level handler
// that closes the whole Tag Manager does not also fire.
function onKeydown(e: KeyboardEvent) {
  if (e.key !== "Escape") return;
  e.stopPropagation();
  emit("cancel");
}

onMounted(() => {
  window.addEventListener("keydown", onKeydown, true);
  void tagsStore.fetchTagUsage(props.sources.map((t) => t.id)).then((u) => (usage.value = u));
  searchInput.value?.focus();
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
  /* Above the Tag Manager modal (z-index 100). */
  z-index: 200;
}
.dlg {
  background: #111;
  border: 1px solid #222;
  border-radius: 12px;
  width: 460px;
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
  gap: 16px;
  overflow: hidden;
}
.dlg-section {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-height: 0;
}
.dlg-label {
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  color: #666;
  font-family: "Inter", sans-serif;
}
.dlg-source-list {
  list-style: none;
  margin: 0;
  padding: 0;
  max-height: 110px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 3px;
}
.dlg-source-list li {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: #ccc;
  font-family: "Inter", sans-serif;
}
.dlg-source-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.dlg-source-meta {
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
.dlg-input {
  background: #1a1a1a;
  border: 1px solid #2a2a2a;
  border-radius: 6px;
  color: #e8e8e8;
  padding: 7px 10px;
  font-size: 13px;
  font-family: "Inter", sans-serif;
  outline: none;
}
.dlg-input:focus {
  border-color: #3a3a3a;
}
.dlg-target-list {
  border: 1px solid #1f1f1f;
  border-radius: 8px;
  max-height: 170px;
  overflow-y: auto;
  padding: 4px;
  background: #0d0d0d;
}
.dlg-target {
  display: flex;
  align-items: center;
  gap: 6px;
  width: 100%;
  background: none;
  border: none;
  border-radius: 6px;
  color: #ccc;
  padding: 5px 8px;
  font-size: 13px;
  font-family: "Inter", sans-serif;
  cursor: pointer;
  text-align: left;
}
.dlg-target:hover {
  background: #1e1e1e;
}
.dlg-target.is-picked {
  background: #14532d;
  color: #e8e8e8;
}
.dlg-target-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.dlg-badge {
  font-size: 9px;
  color: #555;
  background: #1a1a1a;
  padding: 1px 5px;
  border-radius: 4px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}
.dlg-count {
  font-size: 11px;
  color: #555;
}
.dlg-flex {
  flex: 1;
}
.dlg-empty {
  padding: 10px;
  font-size: 12px;
  color: #666;
  font-family: "Inter", sans-serif;
}
.dlg-preview {
  border-top: 1px solid #1a1a1a;
  padding-top: 12px;
  display: flex;
  flex-direction: column;
  gap: 4px;
  font-size: 12px;
  color: #999;
  font-family: "Inter", sans-serif;
}
.dlg-preview b {
  color: #e8e8e8;
  font-weight: 600;
}
.dlg-note {
  color: #666;
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
.dlg-confirm {
  flex: 2;
  background: #22c55e;
  color: #000;
  border: none;
  border-radius: 6px;
  padding: 9px;
  cursor: pointer;
  font-size: 13px;
  font-weight: 500;
  font-family: "Inter", sans-serif;
}
.dlg-confirm:hover:not(:disabled) {
  background: #16a34a;
}
.dlg-confirm.is-armed {
  background: #f59e0b;
}
.dlg-confirm:disabled {
  background: #1f2d23;
  color: #556058;
  cursor: not-allowed;
}
</style>
