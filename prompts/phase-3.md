# Phase 3 — ResourceList UI Overhaul

Wire the column store into ResourceList.svelte: sticky first column, column resize handles, cell alignment, truncation tooltips, compact mode, namespace click handler, and sort persistence.

## First Action

Read `frontend/src/lib/components/ResourceList.svelte` end-to-end — you'll be refactoring this component heavily. Pay attention to how `columns` is currently received as a prop, how `gridTemplateCols` is computed, and how `sortCol`/`sortDir` state works. All of these change to use the column store.

## Context

Phase 1 built the Go backend (types, RPCs, Namespace columns). Phase 2 built the reactive column store that merges descriptor defaults with user prefs. This phase wires that store into the actual ResourceList component, replacing prop-driven columns with store-driven columns and adding the interactive column features: sticky first column, resize handles, alignment, tooltips, compact mode, namespace click, and persistent sort.

Phase 4 (ColumnMenu) runs in parallel with this phase. Coordinate: this phase should NOT touch the `Columns3` button/dropdown — leave the existing sparkline toggle in place. Phase 4 will replace it.

## Files to Read

- `frontend/src/lib/components/ResourceList.svelte` — **what to look for**: `columns` prop, `gridTemplateCols` derived, `sortCol`/`sortDir` state, `ROW_HEIGHT` constant, the virtualizer setup, and cell rendering loop. All of these need modifications.
- `frontend/src/lib/stores/columns.svelte.ts` — **what to look for**: the store API you'll consume: `visibleColumns`, `sortState`, `setSort()`, `resizeColumn()`, `autoFitColumn()`, `compact`. This is the Phase 2 deliverable.
- `frontend/src/routes/ResourceListPage.svelte` — **what to look for**: how it currently passes `columns` prop to ResourceList and manages the GVR lifecycle. You'll change it to initialize the column store for the current GVR instead.
- `frontend/src/lib/stores/cluster.svelte.ts` — **what to look for**: `setSelectedNamespaces()` method signature. Namespace column cells will call this on click.
- `frontend/src/lib/registry/index.ts` — **what to look for**: `defaultAlign()` helper for determining cell alignment from render type.

## Source Documents

- `RESOURCE_LIST_COLUMNS.md` — §ResourceList.svelte section covering sticky column, resize handles, alignment, compact mode, namespace click, sort persistence, and grid template computation.
- `PHASES.md` — Phase 3 section for deliverables, acceptance criteria, and handoff notes.

## What Exists

- `Column` struct with `Align`/`Hidden` fields, Namespace columns on namespaced descriptors (Phase 1)
- ConfigService RPCs for column prefs and compact mode (Phase 1)
- `ColumnDef` with `align?`/`hidden?`, `defaultAlign()` helper (Phase 1)
- `columnStore` singleton with `visibleColumns`, `allColumns`, `sortState`, `setSort()`, `resizeColumn()`, `autoFitColumn()`, `compact`, `loadForGVR()` (Phase 2)
- Current `ResourceList.svelte` receiving `columns: ColumnDef[]` as a prop, with fixed `ROW_HEIGHT = 36`, index-based sort tracking, and no sticky/resize/alignment features

## Deliverables

1. `ResourceList.svelte` refactored to read columns from `columnStore.visibleColumns` instead of a `columns` prop
2. Sticky first column: first cell in header and each body row gets `position: sticky; left: 0; z-index: 10` with `bg-bg` background and `shadow-[2px_0_4px_rgba(0,0,0,0.08)]` (dark: `shadow-[2px_0_4px_rgba(0,0,0,0.3)]`)
3. Column resize handles: thin `<div>` (4px wide) between header cells with `cursor-col-resize`. Mousedown starts tracking, mousemove updates column width live via `columnStore.resizeColumn()`, mouseup finalizes. Double-click triggers auto-fit (measure max content width of visible rows, call `columnStore.autoFitColumn(name, measuredWidth)`)
4. Minimum column width enforced at 20px in resize handler
5. Grid template computed from `columnStore.visibleColumns`: columns with explicit width → `${width}px`, others → `minmax(20px, 1fr)`. Plugin columns and sparkline columns appended after.
6. Cell alignment: each cell gets `text-left`, `text-right`, or `text-center` class based on `col.align ?? defaultAlign(col.renderType)`
7. Cell content wrapped with `title={renderValue(value, col.renderType)}` attribute for truncation tooltip
8. Compact mode: `ROW_HEIGHT` derived from `columnStore.compact` — `28` if compact, `36` otherwise. Passed to virtualizer's `estimateSize`.
9. Namespace column cells: when `col.name === 'Namespace'`, the cell gets an `onclick` handler that calls `clusterStore.setSelectedNamespaces([value])` where `value` is the namespace string
10. Sort state: `sortCol`/`sortDir` replaced with `columnStore.sortState`. `toggleSort()` calls `columnStore.setSort()`. Sort is by column name, not index.
11. `ResourceListPage.svelte` updated: on GVR change, call `columnStore.loadForGVR(gvr)`. Remove `columns` prop passing to ResourceList.

## Tests

- **Frontend test (vitest)**
  - `sticky first column has correct classes` — render ResourceList with mocked store, query first header cell, assert it has `sticky` and `left-0` in its class list
  - `cell alignment matches render type` — render a row with age and text columns, verify age cell has `text-right`, text cell has `text-left`
  - `cell has title attribute` — render a row, query cell content spans, verify each has a `title` attribute matching the rendered value

- **Manual verification**
  - Drag a resize handle between two columns — column width changes live, minimum 20px enforced
  - Double-click a resize handle — column auto-fits to widest visible content
  - Scroll right — first column (Name) stays pinned with visible drop shadow
  - Toggle compact mode — row height visibly decreases from 36px to 28px
  - Click a namespace value in the Namespace column — global namespace filter updates to that single namespace
  - Sort a column, switch to another GVR, switch back — sort preference is preserved

## Acceptance Criteria

- [ ] ResourceList reads columns from `columnStore.visibleColumns`, not a prop
- [ ] First column (Name) is sticky with solid `bg-bg` background and right drop shadow
- [ ] Resize handles appear between header cells; dragging resizes columns live
- [ ] Double-click resize handle auto-fits column to widest visible content
- [ ] Column widths cannot go below 20px
- [ ] Cell text alignment follows `col.align ?? defaultAlign(col.renderType)`
- [ ] Every cell content has a `title` attribute for truncation tooltip
- [ ] Compact mode changes `ROW_HEIGHT` from 36 to 28
- [ ] Clicking a namespace cell sets `clusterStore.setSelectedNamespaces([namespace])`
- [ ] Sort column/direction uses `columnStore.sortState` and persists via `columnStore.setSort()`
- [ ] `ResourceListPage` initializes column store on GVR change
- [ ] `pnpm check` passes
- [ ] Existing frontend tests pass

## Definition of Done

Opening any resource list shows columns from the store. The Name column is pinned when scrolling horizontally, with a subtle drop shadow. Resize handles between header cells allow live column resizing (minimum 20px). Double-clicking a handle auto-fits. Age columns are right-aligned. Hovering truncated cells shows the full value in a browser tooltip. Toggling compact mode shrinks rows. Clicking a namespace value filters to that namespace. Sorting persists when switching between GVRs.

## Known Gotchas

- **Sticky + CSS grid requires explicit background-color.** If the sticky cell has `background: transparent` (the default), scrolled content will show through behind it. Use the `bg-bg` theme token — it works in both light and dark mode. Test by scrolling horizontally with enough columns to overflow.

- **Virtual rows use `transform: translateY()` for positioning.** The virtualizer absolutely positions rows within a relative container. Horizontal scroll is on the parent `scrollContainer`. Sticky works within the horizontal scroll context. If sticky breaks, the fix is to make the inner grid container handle `overflow-x: auto` instead of the outer scroll div — but try the simple approach first.

- **Sort state is now by column name, not index.** The current code uses `sortCol: number` (index into columns array). The store uses `sortState: { column: string; direction }`. When toggling sort, pass the column name. The sort comparator in `filtered` should look up the column by name from `visibleColumns` to get the `expr`.

- **Don't touch the Columns3 button/dropdown.** Phase 4 (ColumnMenu) replaces it. Leave the existing sparkline toggle functional for now. If you remove it, Phase 4 has nothing to replace.

- **Auto-fit measures visible rows only.** For double-click auto-fit, measure the rendered text width of cells in the virtualizer's visible rows only (via `$virtualizer.getVirtualItems()`). This means the widest off-screen value won't be accounted for — that's an acceptable tradeoff per the spec.
