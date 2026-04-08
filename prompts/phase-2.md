# Phase 2 — Frontend Column Store

Create a reactive Svelte 5 store that merges descriptor defaults with user-saved preferences, exposes column visibility/order/width/sort state, and debounce-saves mutations to the Go backend.

## First Action

Read `frontend/src/lib/stores/cluster.svelte.ts` to see the class-based singleton store pattern used throughout this project — your column store must follow the same structure (`$state` fields, exported singleton instance, async initialization method).

## Context

Phase 1 added `Align`/`Hidden` fields to the `Column` struct, column preference types to the config, four new RPCs on `ConfigService`, and Namespace columns on all namespaced descriptors. The Wails bindings are regenerated. This phase builds the reactive frontend layer that consumes those RPCs: a store that loads prefs for the current GVR, merges them with descriptor defaults, and exposes a clean API that ResourceList and ColumnMenu will consume in Phases 3–4.

## Files to Read

- `frontend/src/lib/stores/cluster.svelte.ts` — **what to look for**: the class-based singleton pattern with `$state` fields, async `init()` method, and how reactive state is exposed. Your store follows this exact pattern.
- `frontend/src/lib/registry/index.ts` — **what to look for**: `ColumnDef` interface (now with `align?` and `hidden?`), `DescriptorDef`, `descriptorRegistry.get(gvr)`, and `defaultAlign()` helper. The store uses these to get the full column pool for a GVR.
- `frontend/bindings/github.com/Vilsol/klados/internal/services/configservice.js` — **what to look for**: the generated `GetColumnPrefs`, `SetColumnPrefs`, `GetCompactRows`, `SetCompactRows` function signatures. These are the RPCs your store calls.
- `frontend/src/lib/stores/session.svelte.ts` — **what to look for**: the debounced save pattern (if present), or note that you'll need to implement debouncing yourself (setTimeout/clearTimeout).

## Source Documents

- `RESOURCE_LIST_COLUMNS.md` — §Frontend changes, `columns.svelte.ts` section. Defines the full store API: `visibleColumns`, `allColumns`, `sortState`, and all mutation methods.
- `PHASES.md` — Phase 2 section for deliverables, acceptance criteria, and handoff notes.

## What Exists

- `Column` struct with `Align` and `Hidden` fields (Phase 1)
- `Config.ColumnPrefs` map and `Config.CompactRows` field (Phase 1)
- `ConfigService.GetColumnPrefs(gvr)`, `SetColumnPrefs(gvr, prefs)`, `GetCompactRows()`, `SetCompactRows(compact)` RPCs (Phase 1)
- Generated Wails bindings for the new ConfigService methods (Phase 1)
- `ColumnDef` interface with `align?` and `hidden?` fields, `defaultAlign()` helper (Phase 1)
- Namespace column on every namespaced descriptor with `hidden: true` (Phase 1)
- `descriptorRegistry` that provides `get(gvr)` returning the full `DescriptorDef` including all columns

## Deliverables

1. New file `frontend/src/lib/stores/columns.svelte.ts` exporting a class-based singleton store
2. `loadForGVR(gvr: string)` method — fetches prefs via `ConfigService.GetColumnPrefs(gvr)`, fetches descriptor via `descriptorRegistry.get(gvr)`, merges into reactive state
3. `visibleColumns: ColumnDef[]` — derived from merge: ordered, filtered to visible, with width overrides applied
4. `allColumns: { col: ColumnDef; visible: boolean }[]` — all descriptor columns with visibility flag, for the column menu
5. `sortState: { column: string; direction: 'asc' | 'desc' } | null` — loaded from prefs, updated via `setSort()`
6. `setColumnVisible(name: string, visible: boolean)` — toggles column visibility; Name column cannot be hidden (no-op if attempted)
7. `moveColumn(name: string, direction: 'up' | 'down')` — reorders within visible columns
8. `resizeColumn(name: string, width: number)` — sets explicit width on a column
9. `autoFitColumn(name: string, width: number)` — accepts a pre-computed width from DOM measurement, stores it
10. `setSort(column: string, direction: 'asc' | 'desc')` — updates sort state
11. `reset()` — clears the GVR's prefs entry, reverts to descriptor defaults
12. `save()` — debounced (300-500ms) write to `ConfigService.SetColumnPrefs(gvr, prefs)` on any mutation
13. `compact: boolean` getter/setter — reads/writes via `ConfigService.GetCompactRows` / `SetCompactRows`

## Tests

- **Frontend test (vitest)**
  - `columns store loads descriptor defaults when no prefs exist` — mock `GetColumnPrefs` returning null, call `loadForGVR`, verify `visibleColumns` matches descriptor's non-hidden columns in descriptor order
  - `columns store merges saved prefs with descriptor` — mock prefs with reordered subset and custom widths, verify `visibleColumns` reflects saved order and widths
  - `hidden columns appear in allColumns but not visibleColumns` — descriptor has a column with `hidden: true`, verify it's in `allColumns` with `visible: false` but absent from `visibleColumns`
  - `setColumnVisible hides a column` — hide "Status", verify removed from `visibleColumns`, present in `allColumns` as `visible: false`
  - `setColumnVisible cannot hide Name` — attempt `setColumnVisible('Name', false)`, verify Name remains in `visibleColumns`
  - `moveColumn reorders correctly` — move a middle column up, verify `visibleColumns` order changes
  - `moveColumn up on first column is no-op` — verify order unchanged
  - `resizeColumn updates width` — set width 200 on a column, verify `visibleColumns` entry has `width: 200`
  - `reset clears prefs` — set some prefs, call `reset()`, verify columns revert to descriptor defaults
  - `setSort updates sort state` — call `setSort('Age', 'desc')`, verify `sortState` matches

## Acceptance Criteria

- [ ] `columns.svelte.ts` exports a singleton store class with the API listed above
- [ ] Store correctly merges descriptor columns with saved prefs (order, visibility, widths)
- [ ] Hidden columns appear in `allColumns` but not in `visibleColumns` by default
- [ ] Mutations trigger debounced save to `ConfigService.SetColumnPrefs`
- [ ] `reset()` clears the GVR's prefs and reverts to descriptor defaults
- [ ] Name column cannot be hidden
- [ ] Compact mode getter/setter works via `ConfigService`
- [ ] All vitest tests pass
- [ ] `pnpm check` passes

## Definition of Done

Importing `columnStore` from `columns.svelte.ts` and calling `columnStore.loadForGVR('core.v1.pods')` populates `visibleColumns` with the pods descriptor's non-hidden columns in correct order. Calling `setColumnVisible('Namespace', true)` adds the hidden Namespace column to `visibleColumns`. Calling `reset()` reverts to defaults. All vitest tests pass and `pnpm check` is clean.

## Known Gotchas

- **The store is a singleton, not a factory.** Follow the `clusterStore` pattern: one class instance exported as a module-level constant. When the GVR changes, call `loadForGVR(newGvr)` which reinitializes internal state — don't create a new store instance.

- **`autoFitColumn` must accept a width parameter, not measure DOM itself.** The store has no DOM access. The caller (Phase 3's resize handle double-click handler) measures visible rows and passes the computed width. The store just records it.

- **Plugin columns are NOT managed by this store.** `slotRegistry` handles plugin-injected list columns separately. The store's `allColumns` only includes columns from the descriptor. If you mix them in, the column menu and persistence will break.

- **Debounce save, don't save on every mutation.** Multiple rapid mutations (e.g. dragging a resize handle) should coalesce into one RPC call. Use a 300-500ms debounce with `setTimeout`/`clearTimeout`. Clear the pending timeout in `loadForGVR` to avoid saving stale state for the wrong GVR.

- **`Order` array semantics.** When building prefs to save, `Order` contains only visible column names. When loading: if `Order` is null/empty, derive it from the descriptor column order (filtering out `hidden: true` columns). A column present in the descriptor but absent from both `Order` and `Columns` map = use descriptor default visibility (visible if not `hidden`, hidden if `hidden`).
