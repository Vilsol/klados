# Resource List UX Improvements

**Date:** 2026-05-14
**Scope:** Column management UX + adjacent resource-list improvements
**Primary surfaces:** `frontend/src/lib/components/DataTable.svelte`, `ResourceList.svelte`, `ColumnMenu.svelte` (→ replaced by `ColumnPicker.svelte`), `frontend/src/lib/stores/columns.svelte.ts`, `internal/config/models.go`

## Motivation

Current column management is a single dropdown (`ColumnMenu.svelte`) that mixes three concerns — column visibility, column order, and view options (compact rows, sparklines) — and uses single-step up/down arrow buttons for reorder. Moving a column N positions requires N clicks. There is no direct manipulation on the table header itself.

Adjacent pain points in the resource list:
- The right-click row context menu has minimal items (Delete, Browse Volume, plugin items). There is no quick path to copy a name, view/edit YAML, view logs, or open a terminal — even though all the backing services exist.
- The empty state is plain text ("No resources found") regardless of whether the user has zero resources or has filtered them all out.
- The loading state is plain "Loading…" text rather than skeletons, so layout shifts on first paint.

## Goals

1. Reorder columns by dragging the column header directly in the resource list.
2. Reduce the columns popover to one job: pick which columns are visible.
3. Add direct-manipulation features to the header: right-click menu (sort / auto-fit / hide / pin).
4. Pin the `Name` column (sticky left) so it stays visible during horizontal scroll on wide resources.
5. Expand the row context menu with: Copy name, Copy YAML, View YAML, Edit YAML, View logs, Open terminal.
6. Distinguish "no resources" vs "filtered to zero" in the empty state; offer a "Clear filters" affordance for the latter.
7. Render skeleton rows on initial load instead of plain "Loading…" text.

## Non-goals

- Column presets / saved views.
- Multi-column sort (shift-click).
- Auto-refresh interval / last-refreshed indicator.
- Drag-reorder inside the picker popover (drag happens in the table; the picker is checkboxes only).

## Design

### 1. Reorder via header drag (primary mechanism)

Drag any column header to reorder. Drop indicator (vertical accent-colored line) renders between adjacent headers during the drag.

- Library: **`svelte-dnd-action`** (selected after evaluating `@thisux/sveltednd`, `Dnd-Master`, and HTML5 native — see §11). Mature, action-based, with built-in keyboard support (Tab → Space → ←→ → Space).
- The header `{#each visibleColumns as col}` block in `DataTable.svelte` gets wrapped in `use:dndzone={{items, type: 'columns', flipDurationMs: 150, dragDisabled: pinned}}`. `type: 'columns'` is unique so it never conflicts with row-level DnD if added later.
- Resize handles, sort buttons, and the right-click target inside each header cell are marked with the `data-no-dnd` attribute pattern that `svelte-dnd-action` honors (or wrapped with `dragHandle` strictly limited to the cell label area — final pattern picked during plan based on the cleaner interaction model).
- On `onfinalize`, the new visible order is sent to `columnStore.reorderVisible(names[])`. Persisted per-GVR in the existing prefs blob.

### 2. Pinned columns (sticky left)

`Name` is pinned by default. Users can pin/unpin any column via the header right-click menu.

- `DataTable.svelte` header restructures from a single grid to two siblings in a flex row:
  - **Pinned grid** (`position: sticky; left: 0; z-index: 10`) for pinned columns. Excluded from the dndzone.
  - **Main grid** (the dndzone) for the rest.
- Body rows mirror the structure. The pinned half also uses `position: sticky; left: 0` and inherits the row's background for proper occlusion.
- Per-row prefix snippet (checkbox column) stays in its current leftmost position and is also sticky.

### 3. Column picker (replaces `ColumnMenu`)

New file: `frontend/src/lib/components/ColumnPicker.svelte`. **Visibility only — no ordering.**

```
┌─────────────────────────────────────┐
│ Columns                       Reset │
│ ┌─────────────────────────────────┐ │
│ │ 🔍 Filter…                      │ │
│ └─────────────────────────────────┘ │
│ ☑ Name        (pinned — required)   │
│ ☑ Ready                             │
│ ☑ Status                            │
│ ☐ Restarts                          │
│ ☑ Age                               │
│ ☐ IP                                │
│ ☐ Node                              │
└─────────────────────────────────────┘
```

- Flat checkbox list. Pinned columns rendered first with checkbox disabled (clear visual that they're required visible).
- Filter box (substring match against `col.name`) for resources with many columns (CRDs with `additionalPrinterColumns`).
- Items listed in current visible-order followed by hidden-after.
- When a previously hidden column is enabled, it is appended to the end of `visibleColumns`. The user then drags the header to its preferred position.
- Reset button restores defaults via existing `columnStore.reset()`.

Removed from this popover: compact toggle, sparkline toggles.

### 4. View options menu (split from column menu)

New file: `frontend/src/lib/components/ViewOptionsMenu.svelte`. Triggered by a separate "eye" icon button in the toolbar next to the Columns button. Contains:

- Compact rows checkbox.
- Sparkline toggles (CPU / Memory) — only rendered when `gvr ∈ sparklineGvrs`.

This is a small, focused popover. Separation of concerns is the entire point: the column popover is for "what columns?", the view popover is for "how dense, what extras?".

### 5. Header right-click context menu

New file: `frontend/src/lib/components/HeaderContextMenu.svelte`. Triggered by right-clicking any column header in `DataTable.svelte`. Items:

- **Sort ascending** — calls `onsort(name, 'asc')`.
- **Sort descending** — calls `onsort(name, 'desc')`.
- **Auto-fit** — calls existing `autoFit(name)` (already in DataTable).
- **Pin to left** / **Unpin** — calls `columnStore.setPinned(name, value)`.
- **Hide column** — calls `columnStore.setColumnVisible(name, false)`. Hidden for `Name` (or any pinned-required column).

Existing left-click sort behavior on the header label is unchanged.

### 6. Row context menu expansion

In `ResourceList.svelte`, the existing right-click row menu is expanded:

```
┌─────────────────────────────┐
│ Copy name                   │   always
│ Copy YAML                   │   always
│ View YAML                   │   always (opens kind: "yaml" tab)
│ Edit YAML                   │   if canMutate
├─────────────────────────────┤
│ View logs                   │   if pod-like
│ Open terminal               │   if pod + canMutate
├─────────────────────────────┤
│ Browse volume               │   if PVC (existing)
│ <plugin items>              │   existing
├─────────────────────────────┤
│ Delete                      │   if canMutate, destructive style
└─────────────────────────────┘
```

Wiring:
- **Copy name** — `navigator.clipboard.writeText(item.metadata.name)` + `notificationStore.push('Copied name', 'info')`.
- **Copy YAML** — serialize the item using the existing single-item helper (extract from `exportItems` in `frontend/src/lib/utils/export.ts` into a thin reusable `itemToYaml(item)` in a new `yamlClipboard.ts`), then `clipboard.writeText` + toast.
- **View YAML** — `bottomPanelStore.addTab({kind: "yaml", ...})` (the kind already exists in `PanelKind`).
- **Edit YAML** — same as View YAML but with the panel's edit mode enabled. If the YAML panel doesn't currently expose an `editable` flag in the tab payload, add an optional `editable?: boolean` field to `PanelTab` and have `BottomPanel.svelte` pass it through. This is a small, additive change; defaults to view-only.
- **View logs** —
  - If `gvr === "core.v1.pods"` → `kind: "logs"`.
  - If `gvr ∈ {apps.v1.deployments, apps.v1.statefulsets, apps.v1.daemonsets, apps.v1.replicasets, batch.v1.jobs, batch.v1.cronjobs}` → `kind: "aggregate-logs"`.
  - Otherwise the item is hidden.
- **Open terminal** — only when `gvr === "core.v1.pods"` and `canMutate`. `bottomPanelStore.addTab({kind: "terminal", ...})`.

Pod-like detection uses `gvr` string equality; no runtime inspection of the resource shape.

### 7. Empty state

In `DataTable.svelte`, when `items.length === 0 && !loading`:

```
        ┌──────────────────────────┐
        │       (icon)             │
        │   No pods match filters  │
        │                          │
        │   [ Clear filters ]      │   ← only if searchTerms.length > 0
        └──────────────────────────┘
```

- A new optional snippet `emptyAction?: Snippet` is added to `DataTable.svelte`'s props.
- `ResourceList.svelte` provides the snippet, conditionally rendering a "Clear filters" button when `searchTerms.length > 0`. The button sets `searchQuery = ''; searchTerms = []`.
- The `emptyMessage` prop already exists; `ResourceList.svelte` switches between `"No resources found"` (no filters) and `"No resources match these filters"` (filters set).

### 8. Loading skeleton

Replace the `"Loading…"` text in `DataTable.svelte`'s `{#if loading}` branch with skeleton rows when `loading && items.length === 0`. During incremental refetches (items already exist), the existing items continue to render — no skeleton flash.

- 8 placeholder rows at the configured `rowHeight`.
- Each cell: `<div class="h-3 rounded bg-surface-hover animate-pulse" style="width: 60%"></div>` (width varies 40–80% per cell to avoid mechanical uniformity).
- Pinned-column cells render their own skeleton so the sticky layout doesn't collapse.

### 9. Store API (`columns.svelte.ts`)

Existing API preserved. Additions:

```ts
class ColumnStore {
  // existing
  visibleColumns, allColumns, sortState, compact
  setColumnVisible(name, visible)
  moveColumn(name, "up" | "down")   // retained as programmatic / a11y escape hatch
  resizeColumn(name, width)
  setSort(column, direction)
  reset()
  setCompact(value)

  // new
  reorderVisible(names: string[]): void           // accepts the post-drop full visible order
  setPinned(name: string, pinned: boolean): void  // mutates pinnedNames + reorders so pinned-first
  isPinned(name: string): boolean
  pinnedNames(): string[]                         // derived helper for DataTable's pinned grid
}
```

- `reorderVisible(names[])` replaces `visibleColumns` wholesale. Validates that every passed name is currently visible (defensive), then saves.
- `setPinned(name, true)` moves the column to the front of `visibleColumns` immediately after any other pinned columns (preserving relative pinned order). `setPinned(name, false)` removes it from the pinned set; its index in `visibleColumns` is left where the user pinned it from. Pinned set persists.
- `pinnedNames()` is a derived list from `visibleColumns` filtered against the persisted pinned set.

### 10. Persistence (`GVRColumnPrefs`)

`internal/config/models.go` gains:

```go
type GVRColumnPrefs struct {
    Order   []string                    `json:"order,omitempty"`
    Columns map[string]ColumnSettings   `json:"columns,omitempty"`
    Sort    *SortPrefs                  `json:"sort,omitempty"`
    Pinned  []string                    `json:"pinned,omitempty"`   // NEW
}
```

- Missing `Pinned` defaults to `["Name"]` when applied. The default is applied in `columns.svelte.ts#applyPrefs` (frontend), not in Go, so the Go-side stays additive and backwards-compatible.
- Bindings are regenerated with `wails3 generate bindings`.
- `frontend/src/routes/settings/ColumnSettings.svelte` displays pinned columns alongside the existing order/sort summary (read-only, like the other fields).

### 11. DnD library selection (decision record)

Evaluated:

| Library | Svelte 5 | Keyboard a11y | Touch | Maturity | Verdict |
|---|---|---|---|---|---|
| `svelte-dnd-action` 0.9.69 | ✅ (uses `onconsider`/`onfinalize`) | ✅ built-in | ✅ | ~5 years prod | **Selected** |
| `@thisux/sveltednd` 0.4.1 | ✅ runes-native | ❓ undocumented | ✅ pointer | low (5 npm dependents, pre-1.0) | Rejected — accessibility risk |
| Dnd-Master | ✅ | partial | partial | niche | Rejected — small ecosystem |
| HTML5 native | ✅ | ❌ DIY | ❌ DIY | — | Rejected — re-implements keyboard + screen-reader behavior |

Klados targets keyboard-heavy power users (kubectl audience). The accessibility story of `svelte-dnd-action` is the deciding factor. The "API predates runes" caveat is mild: items are declared `$state`, reassigned inside event handlers; reactivity unaffected.

### 12. Files

**New:**
- `frontend/src/lib/components/ColumnPicker.svelte`
- `frontend/src/lib/components/ViewOptionsMenu.svelte`
- `frontend/src/lib/components/HeaderContextMenu.svelte`
- `frontend/src/lib/utils/yamlClipboard.ts`

**Modified:**
- `frontend/src/lib/components/DataTable.svelte` — header dndzone, pinning split, header ctx menu, `emptyAction` snippet, skeleton rows, `onreorder` event
- `frontend/src/lib/components/ResourceList.svelte` — swap picker, add view-options + view-logs + open-terminal + copy/view/edit YAML to ctx menu, wire `onreorder`, conditional empty message + Clear filters snippet
- `frontend/src/lib/stores/columns.svelte.ts` — `reorderVisible`, `setPinned`, `isPinned`, `pinnedNames`, default-pin Name
- `internal/config/models.go` — add `Pinned []string` to `GVRColumnPrefs`; regen Wails bindings
- `frontend/src/routes/settings/ColumnSettings.svelte` — display pinned columns
- `frontend/src/lib/stores/bottom-panel.svelte.ts` — optional `editable?: boolean` on `PanelTab` (used by Edit YAML)
- `frontend/src/lib/components/BottomPanel.svelte` — pass `editable` through to the YAML panel
- `frontend/package.json` — add `svelte-dnd-action` dependency (via `pnpm add` in `frontend/`)

**Removed (requires explicit confirmation before deletion per repo rule):**
- `frontend/src/lib/components/ColumnMenu.svelte` — replaced by `ColumnPicker.svelte`

### 13. Testing

**Unit (`vitest`):**
- `columnStore.reorderVisible` — reorders, persists, ignores unknown names.
- `columnStore.setPinned` — moves pinned-true to the front; setPinned-false leaves position; persistence round-trip.
- `columnStore.pinnedNames` — derives correctly; defaults to `["Name"]` when no prefs saved.

**Component (`@testing-library/svelte`):**
- `ColumnPicker.svelte` — filter narrows list; toggling a checkbox calls `setColumnVisible`; pinned checkbox is disabled; Reset calls `reset`.
- `ViewOptionsMenu.svelte` — compact toggle calls `setCompact`; sparkline toggle fires `onSparklineToggle` with the right list.
- `HeaderContextMenu.svelte` — each item triggers the right callback; "Hide" not shown for pinned columns.
- `DataTable.svelte` — `onreorder` fires with the new order after a simulated drop; pinned grid contains exactly the pinned columns; main grid the rest; right-click header opens menu; empty state renders the `emptyAction` snippet only when provided; skeleton renders only on initial load.
- `ResourceList.svelte` — "View logs" calls `bottomPanelStore.addTab` with `kind: "logs"` for pods, `"aggregate-logs"` for pod-owners; "Open terminal" only renders for `core.v1.pods` + `canMutate`; "Copy name" writes to clipboard and pushes a notification.

**Integration:**
- Existing tests under `frontend/src/lib/__tests__/` continue to pass (LogsPanel, TerminalPanel, BottomPanel, TabBar, session).

### 14. Migration / risk

- **Pinned-column layout split**. Restructures `grid-template-columns` from one grid to two (pinned + main) inside a flex row. Risk: cell alignment regression, especially when the main grid contains `1fr` columns whose total width was previously computed against the full viewport. Mitigation: keep the change isolated, visual-diff Pods / Deployments / a CRD with many columns before declaring done.
- **`svelte-dnd-action` in Wails WebView (CGO/GTK on Linux)**. The library uses HTML5 DnD events. WebKitGTK supports HTML5 DnD; no known issues in the codebase. Verified during plan by spiking a header reorder before refactoring the full layout.
- **`GVRColumnPrefs.Pinned`**. Additive JSON field. Old configs without `pinned` load fine; the frontend supplies the default `["Name"]`.

## Open questions

None — proceeding to plan.
