# Bulk Operations & List Enhancements — Design Spec

## Scope

1. **Multi-select** — checkbox column in ResourceList, shift+click range select, select-all
2. **Bulk delete** — delete multiple selected resources with progress dialog
3. **Bulk label/annotate** — add/remove/overwrite labels or annotations on selected resources
4. **Bulk scale** — scale Deployments and StatefulSets (set to N / +N / -N)
5. **Filter by annotations** — chip-based annotation filters alongside existing label filters
6. **Export filtered list** — export visible or selected items as YAML or JSON

### Out of scope

- Saved filters / views (future feature, layers on top of this work)
- Custom column definitions (separate feature)

---

## Architecture

### SelectionStore (`frontend/src/lib/stores/selection.svelte.ts`)

Singleton reactive store, decoupled from ResourceStore for plugin extensibility.

**State:**
- `selectedKeys: Set<string>` — key format: `{namespace}/{name}` (namespaced) or `{name}` (cluster-scoped)
- `selectedGVR: string` — GVR of the currently selected items (selection is always homogeneous)
- `selectedItems: Map<string, object>` — cached unstructured objects for selected keys

**API:**
- `toggle(key, item)` / `select(key, item)` / `deselect(key)`
- `selectRange(fromKey, toKey, visibleKeys, items)` — shift+click range select
- `selectAll(visibleKeys, items)` / `deselectAll()`
- `clear()` — full reset
- `isSelected(key): boolean`
- `count: number` — derived
- `items(): object[]` — returns selected unstructured objects

**Lifecycle:**
- Namespace change → `clear()`
- Navigation away from resource list page → `clear()`
- GVR change → `clear()`

### Selection behavior

- Selection preserved across filter changes (name search, label filter, annotation filter)
- Selection cleared on namespace change or GVR navigation
- Plugins can read/write selection via the store's public API

---

## Components

### Checkbox Column (ResourceList.svelte)

- First column, ~36px wide, not sortable, not hideable
- **Header cell:** select-all checkbox (tri-state: unchecked / indeterminate / checked)
- **Row cell:** checkbox per row, bound to `selectionStore.isSelected(key)`
- **Click:** `selectionStore.toggle(key, item)`
- **Shift+click:** `selectionStore.selectRange(...)` using visible row order
- **Read-only mode:** checkboxes hidden when `clusterStore.canMutate()` is false

### BulkActionBar.svelte

Floating bar at bottom center of viewport, rendered in the layout layer (not inside ResourceList).

**Visibility:** slide-up transition when `selectionStore.count > 0`.

**Layout:**
- Left: "**N selected**" or "**N selected (M not visible)**" when filtered items are selected + "Clear" button
- Right: action buttons, contextually shown by GVR:
  - **Delete** — always
  - **Label** — always
  - **Annotate** — always
  - **Scale** — only for `apps.v1.deployments` and `apps.v1.statefulsets`
  - **Export** — always (YAML/JSON dropdown)

"Not visible" count: ResourceList exposes its current visible (filtered/sorted) keys as a reactive signal (e.g. `visibleKeys: Set<string>`). BulkActionBar derives the not-visible count by diffing `selectionStore.selectedKeys` against this set. This can be a writable signal on the SelectionStore that ResourceList updates whenever its filtered view changes.

**Plugin extensibility:** plugins register bulk actions via the existing action registration API. The bar queries registered actions for the current GVR and appends them after built-in actions.

---

## Bulk Operation Dialogs

### Bulk Delete

- Confirmation dialog lists all selected resources (name, namespace)
- Calls `ResourceService.DeleteResource()` sequentially per item
- Dialog stays open showing per-item progress (checkmarks / error icons)
- Toast summary on completion: "Deleted 5/5 resources" or "Deleted 3/5 — 2 failed"
- Successful items deselected; failed items remain selected for retry

### Bulk Label / Bulk Annotate

- Shared dialog component with mode prop: `"labels"` | `"annotations"`
- Shows current common values across selection (intersection of keys)
- Actions: add key=value, remove key, overwrite existing key
- Each change is a `Patch()` call (JSON merge patch) per resource
- Same progress/error pattern as delete

### Bulk Scale

- Mode toggle: "Set to" / "Increase by" / "Decrease by"
- Single numeric input
- Preview list: each resource shows current replicas → target replicas
- "Decrease by" floors at 0
- Calls `ResourceService.ScaleResource()` per item
- Same progress/error pattern as delete

### Error handling (all operations)

Failed items remain selected after the operation so the user can retry. Successful items are deselected and removed from the dialog's progress list.

---

## Annotation Filter

**UI:** "Add annotation filter" button next to existing label filter, matching the chip pattern.

**Interaction:**
- Click → popover with key + value inputs
- Each filter renders as a removable chip: `annotation:key=value`
- Label chips and annotation chips visually distinguishable (different color/prefix)

**Filtering logic:**
- Annotation filters are AND-ed together
- AND-ed with label filters and name search
- Client-side filtering on `metadata.annotations` from the unstructured objects
- No backend changes needed

---

## Export

**Two entry points:**
- Filter bar area — always visible, exports all currently visible (filtered) items
- Floating bulk action bar — exports only selected items

**Formats:**
- **YAML** — multi-document with `---` separators
- **JSON** — array of resource objects

**Delivery:** browser file download. Filename: `{gvr}-{timestamp}.yaml` or `.json`.

**Implementation:** client-side only. Uses `js-yaml` (existing dependency) for YAML, `JSON.stringify` for JSON. No backend call needed.

---

## Backend Changes

Minimal. All existing backend operations support the bulk use case:
- `ResourceService.DeleteResource()` — called per item
- `ResourceService.ScaleResource()` — called per item
- `ResourceService.Patch()` — called per item for label/annotation changes

No new backend endpoints needed. Bulk operations are orchestrated client-side with sequential calls and progress tracking.

---

## Key Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Selection state location | Dedicated SelectionStore | Plugin extensibility, decoupled from ResourceStore |
| Checkbox visibility | Always visible (not mode-based) | More discoverable, no hidden functionality |
| Bulk action bar placement | Floating bottom bar | Visually distinct, doesn't compete with filters |
| Selection on filter change | Preserved | Users filter then bulk act on visible set |
| Selection on namespace change | Cleared | Namespace switch is a major context change |
| Bulk scale modes | Set to / +N / -N | Covers both absolute and relative use cases simply |
| Annotation filter pattern | Chips (like labels) | Extensible pattern for future filter types |
| Export scope | Visible + selected (two entry points) | Covers both use cases without ambiguity |
