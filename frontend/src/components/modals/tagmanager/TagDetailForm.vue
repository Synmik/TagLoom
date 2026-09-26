<template>
  <div class="tag-form" @keydown.ctrl.enter.prevent="save" @keydown.meta.enter.prevent="save">
    <div class="form-group">
      <label>Name</label>
      <input
        ref="nameInput"
        v-model="form.name"
        placeholder="Tag name"
        class="form-input"
        :class="{ 'input-error': nameError }"
        @keyup.enter="save"
      />
      <span v-if="nameError" class="error-text">{{ nameError }}</span>
      <!-- Names are unique, so a collision is usually the tag the user meant. -->
      <button
        v-if="duplicateTag && isEditing"
        type="button"
        class="link-btn"
        @click="duplicateTag && emit('mergeInto', duplicateTag.id)"
      >
        Merge “{{ tag?.name }}” into “{{ duplicateTag.name }}” instead
      </button>
    </div>

    <div class="form-group">
      <label>Color</label>
      <ColorPicker v-model="form.color" />
    </div>

    <div class="form-group">
      <label>Parent Tag</label>
      <select v-model="form.parentId" class="form-select" :disabled="parentLocked">
        <option :value="null">None (root)</option>
        <option v-for="candidate in parentTags" :key="candidate.id" :value="candidate.id">
          {{ candidate.name }}
        </option>
      </select>
      <span v-if="parentLocked" class="hint-text">
        Categories are top-level tags<span v-if="willMoveToTopLevel">
          — saving moves this one up</span
        >
      </span>
    </div>

    <div class="form-group">
      <label>Aliases</label>
      <div class="aliases-list">
        <div v-for="(alias, idx) in form.aliasList" :key="idx" class="alias-chip">
          <span>{{ alias }}</span>
          <button type="button" class="alias-remove" @click="removeAlias(idx)">
            <X :size="12" />
          </button>
        </div>
        <span v-if="form.aliasList.length === 0" class="aliases-empty">No aliases</span>
      </div>
      <input
        v-model="newAlias"
        placeholder="Add alias and press Enter"
        class="form-input"
        @keyup.enter="addAlias"
      />
    </div>

    <div class="form-group checkbox-group">
      <label class="checkbox-label">
        <input v-model="form.isCategory" type="checkbox" />
        <span class="checkbox-text">
          Is Category
          <span class="checkbox-hint">Can only be a parent — not assignable to files</span>
        </span>
      </label>
    </div>

    <!-- Turning a used tag into a category is allowed; its links are not touched. -->
    <div v-if="categoryKeepsFiles" class="warn-box">
      Already on <b>{{ directFileCount }}</b> file(s). New assignments will be blocked, existing
      links stay.
      <button type="button" class="link-btn" @click="tag && emit('showFiles', tag.id)">
        Show those {{ directFileCount }} files
      </button>
    </div>

    <div v-if="isEditing" class="usage-line">
      Used by <b>{{ directFileCount }}</b> file(s) · {{ childTagCount }} child tag(s) ·
      {{ form.aliasList.length }} alias(es)
      <span v-if="underItCount > 0">· {{ underItCount }} file(s) under it</span>
    </div>

    <div v-if="showActions" class="form-actions">
      <button class="save-btn" :disabled="!canSave" @click="save">
        {{ isEditing ? "Save" : "Create" }}
      </button>
      <button v-if="isEditing && showDelete" class="delete-btn" @click="emit('deleteRequest')">
        Delete
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from "vue";
import { X } from "@lucide/vue";
import ColorPicker from "../../common/ColorPicker.vue";
import { useTagsStore } from "../../../stores/tags";
import { useToast } from "../../../composables/useToast";
import type { Tag, TagCreate, TagUpdate } from "../../../types/tag";

const props = withDefaults(
  defineProps<{
    /** Tag being edited. Null/undefined = create mode. */
    tag?: Tag | null;
    /** Pre-selected parent for create mode (e.g. the category that was selected in the manager). */
    presetParentId?: number | null;
    /** Create a category rather than a plain tag. */
    presetCategory?: boolean;
    /** Render the Save/Create (and Delete) buttons. False when the host renders its own footer. */
    showActions?: boolean;
    /** Render the Delete button. False when the host owns deletion (Tag Manager). */
    showDelete?: boolean;
  }>(),
  { tag: null, presetParentId: null, presetCategory: false, showActions: true, showDelete: true },
);

const emit = defineEmits<{
  saved: [tag: Tag | null];
  deleteRequest: [];
  /** Filter the gallery by this tag (the category warning's jump-out). */
  showFiles: [tagId: number];
  /** Merge the tag being edited into the one whose name was typed. */
  mergeInto: [tagId: number];
}>();

const tagsStore = useTagsStore();
const { success, error: toastError } = useToast();

interface FormState {
  name: string;
  color: string;
  parentId: number | null;
  aliasList: string[];
  isCategory: boolean;
}

const form = ref<FormState>(blankForm());
const newAlias = ref("");
const nameInput = ref<HTMLInputElement | null>(null);
/** Snapshot of the form as last loaded/saved — drives `isDirty`. */
let baseline = JSON.stringify(blankForm());
/** The parent to go back to when "Is Category" is unchecked again.
 *  Declared here, not next to the watcher: `loadFrom` clears it, and the watcher
 *  that loads runs during setup, before a later declaration would be initialised. */
let parentBeforeCategory: number | null = null;

const isEditing = computed(() => !!props.tag);

function blankForm(): FormState {
  return { name: "", color: "", parentId: null, aliasList: [], isCategory: false };
}

async function loadFrom(tag: Tag | null): Promise<void> {
  // A previous tag's remembered parent must not follow this one.
  parentBeforeCategory = null;
  newAlias.value = "";

  if (!tag) {
    const fresh = blankForm();
    fresh.parentId = props.presetCategory ? null : (props.presetParentId ?? null);
    fresh.isCategory = props.presetCategory ?? false;
    form.value = fresh;
    baseline = JSON.stringify(form.value);
    return;
  }

  // Fill every field the tag answers on its own, before asking for aliases. An
  // alias lookup that fails or lags must never leave the form — and so the name,
  // which Save requires — empty.
  form.value = {
    name: tag.name,
    color: tag.color || "",
    // A category is a top-level tag, so a category that still has a parent
    // (older data) is shown at the top level; saving moves it there.
    parentId: tag.is_category === 1 ? null : (tag.parent_id ?? null),
    aliasList: [],
    isCategory: tag.is_category === 1,
  };
  baseline = JSON.stringify(form.value);

  const tagID = tag.id;
  try {
    const aliases = (await tagsStore.getTagAliases(tagID)) ?? [];
    // The user may have moved on to another tag while this was in flight.
    if (props.tag?.id !== tagID) return;
    const editedAlready = JSON.stringify(form.value) !== baseline;
    form.value.aliasList = aliases;
    // Aliases land after the snapshot, so they are not an unsaved edit — unless
    // the user has since touched the form, in which case keep them dirty.
    if (!editedAlready) baseline = JSON.stringify(form.value);
  } catch (e) {
    toastError(`Could not load aliases: ${e instanceof Error ? e.message : String(e)}`);
  }
}

watch(
  () => props.tag,
  (tag) => {
    void loadFrom(tag ?? null);
  },
  { immediate: true },
);

// A category groups other tags and is never one itself, so its parent is not
// selectable — and any parent it still has is cleared.
const parentLocked = computed(() => form.value.isCategory);

watch(
  () => form.value.isCategory,
  (isCategory, wasCategory) => {
    if (isCategory) {
      if (!wasCategory && form.value.parentId != null) {
        parentBeforeCategory = form.value.parentId;
      }
      form.value.parentId = null;
      return;
    }
    // Restore only while the picker still says "(top level)": a parent chosen
    // while the box was checked is a choice, not something to undo.
    if (wasCategory && form.value.parentId == null) {
      form.value.parentId = parentBeforeCategory;
      parentBeforeCategory = null;
    }
  },
);

const addAlias = () => {
  const val = newAlias.value.trim().toLowerCase();
  if (!val) return;
  if (form.value.aliasList.includes(val)) {
    newAlias.value = "";
    return;
  }
  form.value.aliasList.push(val);
  newAlias.value = "";
};

const removeAlias = (idx: number) => {
  form.value.aliasList.splice(idx, 1);
};

// Candidates for the parent dropdown: every tag except this one and its own
// descendants — either would create a cycle the tag tree cannot render.
const parentTags = computed(() => {
  const allTags = tagsStore.tags || [];
  const self = props.tag;
  if (!self) return allTags;
  const excluded = new Set<number>([self.id, ...tagsStore.getAllDescendantIds(self.id)]);
  return allTags.filter((t) => !excluded.has(t.id));
});

// Case-insensitive name collision ("App" vs "app") — tag names are unique
// case-insensitively in the DB (idx_tags_name_nocase).
const duplicateTag = computed<Tag | null>(() => {
  const name = form.value.name.trim().toLowerCase();
  if (!name) return null;
  return (
    (tagsStore.tags || []).find(
      (t) => t.name.toLowerCase() === name && t.id !== (props.tag?.id ?? -1),
    ) ?? null
  );
});

const nameError = computed(() =>
  duplicateTag.value ? `Tag "${duplicateTag.value.name}" already exists (case-insensitive)` : "",
);

/* Usage of the tag being edited — the numbers the save decision rests on. */
const directFileCount = computed(() => (props.tag ? (tagsStore.tagCounts[props.tag.id] ?? 0) : 0));
const childTagCount = computed(() =>
  props.tag ? tagsStore.getAllDescendantIds(props.tag.id).length : 0,
);
const underItCount = computed(() =>
  props.tag && props.tag.is_category === 1 ? tagsStore.getAggregateCount(props.tag.id) : 0,
);
const categoryKeepsFiles = computed(
  () => form.value.isCategory === true && directFileCount.value > 0,
);
/** A nested tag that is becoming a category loses its parent on save. */
const willMoveToTopLevel = computed(() => {
  if (!form.value.isCategory) return false;
  const originalParent = props.tag ? props.tag.parent_id : props.presetParentId;
  return originalParent != null;
});

const isDirty = computed(() => JSON.stringify(form.value) !== baseline);
const canSave = computed(() => form.value.name.trim().length > 0 && !nameError.value);

function buildPayload(): TagCreate {
  return {
    name: form.value.name.trim(),
    color: form.value.color,
    parent_id: form.value.parentId ?? undefined,
    is_category: form.value.isCategory ? 1 : 0,
    sort_order: 0,
    aliases: [...form.value.aliasList],
  };
}

/**
 * Validate and persist. Resolves to the saved tag's name, or null when the
 * form is invalid or the save failed (the failure is surfaced as a toast).
 */
async function save(): Promise<string | null> {
  if (!canSave.value) return null;

  const payload = buildPayload();
  const tagName = payload.name;

  try {
    if (props.tag) {
      const update: TagUpdate = { id: props.tag.id, ...payload };
      // `parent_id` is an optional number in the bindings, so "no parent" reaches
      // the backend as "leave it alone". Clearing one is a move to the top level.
      const moveToTopLevel =
        !form.value.isCategory && props.tag.parent_id != null && form.value.parentId === null;
      await tagsStore.updateTag(update);
      if (moveToTopLevel) await tagsStore.moveTag(props.tag.id, null);
      success(`Tag "${tagName}" updated`);
    } else {
      await tagsStore.createTag(payload);
      success(`Tag "${tagName}" created`);
    }
  } catch (e: any) {
    toastError("Failed to save tag: " + (e.message || String(e)));
    return null;
  }

  baseline = JSON.stringify(form.value);
  emit("saved", props.tag ?? null);
  return tagName;
}

function focusName(): void {
  nameInput.value?.focus();
}

defineExpose({ save, isDirty, canSave, focusName });
</script>

<style scoped>
.tag-form {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.form-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.form-group label {
  color: #666;
  font-size: 10px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}
.form-input,
.form-select {
  background: #1a1a1a;
  border: 1px solid #2a2a2a;
  color: #e8e8e8;
  border-radius: 6px;
  padding: 7px 10px;
  font-size: 13px;
  font-family: "Inter", sans-serif;
  outline: none;
  transition: border-color 0.15s;
}
.form-input:focus,
.form-select:focus {
  border-color: #22c55e;
}
.form-input::placeholder {
  color: #444;
}
.form-select:disabled {
  color: #555;
  cursor: not-allowed;
}
.form-select option {
  background: #1a1a1a;
  color: #e8e8e8;
}
.input-error {
  border-color: #ef4444;
}
.error-text {
  color: #ef4444;
  font-size: 11px;
  margin-top: 2px;
}
.hint-text {
  color: #666;
  font-size: 11px;
}
.warn-box {
  font-size: 11px;
  line-height: 1.5;
  color: #f59e0b;
  background: rgba(245, 158, 11, 0.08);
  border: 1px solid rgba(245, 158, 11, 0.22);
  border-radius: 6px;
  padding: 8px;
  font-family: "Inter", sans-serif;
}
.warn-box b {
  color: #fbbf24;
}
.link-btn {
  background: none;
  border: none;
  color: #4ade80;
  font-size: 11px;
  font-family: "Inter", sans-serif;
  padding: 0;
  text-align: left;
  cursor: pointer;
}
.link-btn:hover {
  text-decoration: underline;
}
.usage-line {
  font-size: 11px;
  color: #777;
  font-family: "Inter", sans-serif;
}
.usage-line b {
  color: #ccc;
}
.aliases-list {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  min-height: 30px;
  background: #1a1a1a;
  border: 1px solid #2a2a2a;
  border-radius: 6px;
  padding: 6px 8px;
  transition: border-color 0.15s;
}
.alias-chip {
  display: flex;
  align-items: center;
  gap: 4px;
  background: #2a2a2a;
  border-radius: 4px;
  padding: 3px 8px;
  font-size: 12px;
  color: #ccc;
}
.alias-chip span {
  max-width: 120px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.aliases-empty {
  color: #555;
  font-size: 12px;
}
.alias-remove {
  background: none;
  border: none;
  color: #666;
  cursor: pointer;
  padding: 0;
  display: flex;
  align-items: center;
  line-height: 1;
  transition: color 0.15s;
}
.alias-remove:hover {
  color: #ef4444;
}
.form-actions {
  display: flex;
  gap: 8px;
  margin-top: 4px;
  padding-top: 12px;
  border-top: 1px solid #1a1a1a;
}
.save-btn {
  flex: 1;
  background: #22c55e;
  color: #000;
  border: none;
  border-radius: 6px;
  padding: 9px;
  cursor: pointer;
  font-size: 13px;
  font-weight: 500;
  font-family: "Inter", sans-serif;
  transition: background 0.15s;
}
.save-btn:hover {
  background: #16a34a;
}
.save-btn:disabled {
  background: #1f2d23;
  color: #556058;
  cursor: not-allowed;
}
.delete-btn {
  background: rgba(239, 68, 68, 0.1);
  color: #ef4444;
  border: 1px solid rgba(239, 68, 68, 0.2);
  border-radius: 6px;
  padding: 9px 14px;
  cursor: pointer;
  font-size: 13px;
  font-family: "Inter", sans-serif;
  transition: all 0.15s;
}
.delete-btn:hover {
  background: rgba(239, 68, 68, 0.2);
}
.checkbox-group {
  margin-top: 2px;
}
.checkbox-label {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  cursor: pointer;
  text-transform: none !important;
  font-size: 13px;
  color: #ccc;
}
.checkbox-label input[type="checkbox"] {
  accent-color: #22c55e;
  margin-top: 2px;
  flex-shrink: 0;
}
.checkbox-text {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.checkbox-text > span {
  display: flex;
  flex-direction: column;
}
.checkbox-hint {
  font-size: 10px;
  color: #666;
  font-weight: 400;
  text-transform: none;
  letter-spacing: 0;
}
</style>
