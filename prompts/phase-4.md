# Phase 4 — Column Menu Component

Build a dropdown component that replaces the sparkline-only toggle with full column management: visibility checkboxes, up/down reorder, compact mode toggle, sparkline toggles, and a reset button.

## First Action

Read the wireframe in `RESOURCE_LIST_COLUMNS.md` (§ColumnMenu.svelte) — it shows the exact layout: checked columns at top with up/down arrows, unchecked at bottom, compact toggle, sparklines section, and reset button. This is your visual spec.

## Context

Phase 1 built the Go backend. Phase 2 built the column store with `allColumns`, `setColumnVisible()`, `moveColumn()`, and `reset()`. This phase builds the UI that calls those store methods. It runs in parallel with Phase 3 (ResourceList UI overhaul) — do NOT assume Phase 3 changes are present in ResourceList.svelte.

The existing `Columns3` button in ResourceList.svelte opens a small sparkline toggle dropdown. This phase replaces that entire dropdown with the new ColumnMenu component. However, since Phase 3 may not be merged yet, implement ColumnMenu as a standalone component that receives the store and sparkline props, and can be dropped into ResourceList by either Phase 3 or a later integration step.

## Files to Read

- `frontend/src/lib/components/ResourceList.svelte` — **what to look for**: the existing `Columns3` button, `columnMenuOpen` state, `availableSparklineCols`, and `toggleSparklineCol()`. Your component replaces all of this. Note the `onclick={(e) => e.stopPropagation()}` pattern on the dropdown.
- `frontend/src/lib/stores/columns.svelte.ts` — **what to look for**: `allColumns` (array of `{ col, visible }`), `setColumnVisible()`, `moveColumn()`, `reset()`, `compact` getter/setter. These are the store methods your UI calls.
- `frontend/src/lib/registry/index.ts` — **what to look for**: `ColumnDef` type shape (name, renderType, etc.) used to display column info in the menu.

## Source Documents

- `RESOURCE_LIST_COLUMNS.md` — §ColumnMenu.svelte wireframe and complete behavior spec. Shows the exact layout, checkbox/button behavior, and sparkline section rules.
- `PHASES.md` — Phase 4 section for deliverables, acceptance criteria, and handoff notes.

## What Exists

- `Column` struct with `Align`/`Hidden` fields, Namespace columns on namespaced descriptors (Phase 1)
- ConfigService RPCs for column prefs and compact mode (Phase 1)
- `columnStore` with `allColumns`, `setColumnVisible()`, `moveColumn()`, `reset()`, `compact` (Phase 2)
- Existing `Columns3` button/dropdown in ResourceList.svelte that only toggles sparkline columns
- `sparklineGvrs`, `sparklineColumns`, `onSparklineToggle` props on ResourceList

## Deliverables

1. New file `frontend/src/lib/components/ColumnMenu.svelte` — self-contained dropdown component
2. Props: `gvr: string`, `sparklineGvrs: string[]`, `sparklineColumns: string[]`, `onSparklineToggle: (columns: string[]) => void`
3. **Columns section**: header row with "Columns" label and "Reset" button. List of all columns from `columnStore.allColumns`. Visible columns at top in current order, hidden columns at bottom. Each row: checkbox + column name + up/down arrow buttons.
4. **Name column**: checkbox always checked and `disabled` — cannot be hidden or reordered
5. **Reorder buttons**: up arrow moves column earlier in order, down arrow moves later. Disabled at boundaries (first visible column can't go up, last can't go down)
6. **Compact mode**: checkbox labeled "Compact rows" that toggles `columnStore.compact`
7. **Sparklines section**: rendered only when `sparklineGvrs.includes(gvr)`. Section header "Sparklines", checkboxes for "CPU" and "Memory" calling `onSparklineToggle`
8. **Reset button**: calls `columnStore.reset()`, reverting all column prefs to descriptor defaults
9. Dropdown positioning: absolute, anchored to the trigger button, right-aligned, with `z-50`

## Tests

- **Frontend test (vitest)**
  - `ColumnMenu renders all columns` — provide a store with 5 columns (1 hidden), render ColumnMenu, query all column name labels, verify all 5 present
  - `Name column checkbox is disabled` — query the checkbox for "Name", assert `disabled` attribute is present and `checked` is true
  - `toggling visibility calls setColumnVisible` — spy on `columnStore.setColumnVisible`, click a column's checkbox, verify spy called with correct args
  - `up button disabled for first column` — query the up arrow for the first visible column, assert it has `disabled` attribute
  - `down button disabled for last visible column` — query the down arrow for the last visible column, assert `disabled`
  - `reset button calls store.reset` — spy on `columnStore.reset`, click Reset button, verify spy called
  - `sparkline section hidden when GVR not in sparklineGvrs` — render with `gvr='core.v1.configmaps'` and `sparklineGvrs=['core.v1.pods']`, verify no sparkline checkboxes rendered

## Acceptance Criteria

- [ ] ColumnMenu is a standalone `.svelte` component that can be imported and rendered
- [ ] All descriptor columns (including hidden) appear in the menu
- [ ] Name column checkbox is always checked and disabled
- [ ] Checking/unchecking a column calls `setColumnVisible` and the column appears/disappears from the list
- [ ] Up/down buttons call `moveColumn` and are disabled at boundaries
- [ ] Compact mode checkbox toggles `columnStore.compact`
- [ ] Sparkline toggles appear only when `sparklineGvrs.includes(gvr)`
- [ ] Reset button calls `columnStore.reset()`
- [ ] `pnpm check` passes
- [ ] Vitest tests pass

## Definition of Done

Rendering `<ColumnMenu gvr="core.v1.pods" ... />` shows a dropdown listing all pod columns (Name, Namespace, Ready, Status, Restarts, Age, plus any Phase 5 hidden columns if merged). Name is locked. Unchecking "Restarts" removes it from `visibleColumns`. Clicking up/down arrows reorders columns. The compact checkbox toggles row height mode. Reset reverts everything. Sparkline toggles appear for pods but not for configmaps.

## Known Gotchas

- **The dropdown must stop click propagation.** Without `onclick={(e) => e.stopPropagation()}` on the dropdown container, clicking checkboxes or buttons inside will bubble up and close the dropdown (the existing pattern uses a window click listener to close). Follow the same pattern as the current sparkline dropdown.

- **Visible columns at top, hidden at bottom.** Don't just render `allColumns` in one list. Split into two groups: visible (in their current order) and hidden (in descriptor order). The up/down buttons only operate on visible columns.

- **Don't assume Phase 3 changes exist in ResourceList.** Since Phases 3 and 4 are parallel, ResourceList may still have the old `columns` prop and sparkline toggle. Build ColumnMenu as a standalone component. It can be integrated into ResourceList in Phase 3 or after both phases merge. Export it and document how to drop it in.

- **The existing sparkline toggle logic must be fully replaceable.** Your component subsumes the sparkline toggle functionality. When ColumnMenu is integrated (either by Phase 3 or later), the old `columnMenuOpen`, `availableSparklineCols`, and `toggleSparklineCol` in ResourceList should be deleted entirely — not kept alongside.
